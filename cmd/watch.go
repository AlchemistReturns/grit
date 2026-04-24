package cmd

import (
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/alchemistreturns/grit/internal/analysis"
	"github.com/alchemistreturns/grit/internal/config"
	"github.com/alchemistreturns/grit/internal/prompt"
	"github.com/alchemistreturns/grit/internal/store"
	"github.com/charmbracelet/lipgloss"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch files for complexity and naming friction",
	RunE:  runWatch,
}

func init() {
	rootCmd.AddCommand(watchCmd)
}

type watchEventType int

const (
	evtFileChange watchEventType = iota
	evtAIReflect
	evtPaste
	evtUndoSpike
	evtDeadTime
	evtTick
)

type watchEvent struct {
	Type       watchEventType
	Path       string
	LinesDelta int
}

type debouncer struct {
	mu     sync.Mutex
	timers map[string]*time.Timer
	delay  time.Duration
	fn     func(path string)
}

func newDebouncer(delay time.Duration, fn func(string)) *debouncer {
	return &debouncer{timers: make(map[string]*time.Timer), delay: delay, fn: fn}
}

func (d *debouncer) trigger(path string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if t, ok := d.timers[path]; ok {
		t.Reset(d.delay)
		return
	}
	d.timers[path] = time.AfterFunc(d.delay, func() {
		d.mu.Lock()
		delete(d.timers, path)
		d.mu.Unlock()
		d.fn(path)
	})
}

func addDirsRecursive(w *fsnotify.Watcher, root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" {
				return filepath.SkipDir
			}
			return w.Add(path)
		}
		return nil
	})
}

var (
	complexStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	pathStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("75"))
)

// pathState tracks per-file activity for dead-time detection.
type pathState struct {
	firstWrite time.Time
	writeCount int
	alerted    bool
}

func runWatch(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := config.EnsureGritDir(); err != nil {
		return err
	}
	db, err := store.Open(config.DBPath())
	if err != nil {
		return err
	}
	defer db.Close()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	cwd, _ := os.Getwd()
	if err := addDirsRecursive(watcher, cwd); err != nil {
		return err
	}

	eventCh := make(chan watchEvent, 32)
	prevContent := make(map[string]string)
	var prevMu sync.Mutex
	pathStates := make(map[string]*pathState)
	deadTimeThreshold := time.Duration(cfg.Thresholds.DeadTimeMinutes) * time.Minute

	extSet := make(map[string]struct{})
	for _, e := range cfg.Watch.Extensions {
		extSet[e] = struct{}{}
	}

	debounceFn := newDebouncer(200*time.Millisecond, func(path string) {
		eventCh <- watchEvent{Type: evtFileChange, Path: path}
	})

	// Ticker for periodic dead-time checks
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			eventCh <- watchEvent{Type: evtTick}
		}
	}()

	go func() {
		for {
			select {
			case ev, ok := <-watcher.Events:
				if !ok {
					return
				}
				if ev.Has(fsnotify.Create) {
					if info, err := os.Stat(ev.Name); err == nil && info.IsDir() {
						watcher.Add(ev.Name)
					}
				}
				if ev.Has(fsnotify.Write) || ev.Has(fsnotify.Create) {
					ext := filepath.Ext(ev.Name)
					if _, ok := extSet[ext]; ok {
						debounceFn.trigger(ev.Name)
					}
				}
			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()

	fmt.Printf("%s  watching %s\n", complexStyle.Render("grít"), pathStyle.Render(cwd))

	for ev := range eventCh {
		switch ev.Type {
		case evtTick:
			for path, state := range pathStates {
				if !state.alerted && state.writeCount > 1 && time.Since(state.firstWrite) >= deadTimeThreshold {
					state.alerted = true
					eventCh <- watchEvent{Type: evtDeadTime, Path: path}
				}
			}

		case evtFileChange:
			processFileChange(db, ev.Path, cfg, prevContent, &prevMu, pathStates, eventCh, cwd)

		case evtAIReflect:
			relPath, _ := filepath.Rel(cwd, ev.Path)
			q := fmt.Sprintf("You added %d lines to %s — was this AI-assisted?", ev.LinesDelta, relPath)
			r := prompt.Ask(q, 30)
			if !r.Skipped && r.Answer != "" {
				id, _ := store.InsertEvent(db, "ai_reflect", false, "")
				store.InsertAnswer(db, id, q, r.Answer, "")
			} else {
				store.InsertEvent(db, "ai_reflect", true, "")
			}

		case evtPaste:
			relPath, _ := filepath.Rel(cwd, ev.Path)
			q := fmt.Sprintf("You pasted %d lines into %s — do you understand what it does?", ev.LinesDelta, relPath)
			r := prompt.Ask(q, 30)
			if !r.Skipped && r.Answer != "" {
				id, _ := store.InsertEvent(db, "paste", false, "")
				store.InsertAnswer(db, id, q, r.Answer, "")
			} else {
				store.InsertEvent(db, "paste", true, "")
			}

		case evtUndoSpike:
			relPath, _ := filepath.Rel(cwd, ev.Path)
			q := fmt.Sprintf("You deleted %d lines from %s — wrong turn or intentional cleanup?", ev.LinesDelta, relPath)
			r := prompt.Ask(q, 30)
			if !r.Skipped && r.Answer != "" {
				id, _ := store.InsertEvent(db, "undo_spike", false, "")
				store.InsertAnswer(db, id, q, r.Answer, "")
			} else {
				store.InsertEvent(db, "undo_spike", true, "")
			}

		case evtDeadTime:
			relPath, _ := filepath.Rel(cwd, ev.Path)
			q := fmt.Sprintf("You've been in %s for %d+ minutes — is the problem clear yet?",
				relPath, cfg.Thresholds.DeadTimeMinutes)
			r := prompt.Ask(q, 30)
			if !r.Skipped && r.Answer != "" {
				id, _ := store.InsertEvent(db, "dead_time", false, "")
				store.InsertAnswer(db, id, q, r.Answer, "")
			} else {
				store.InsertEvent(db, "dead_time", true, "")
			}
		}
	}
	return nil
}

func processFileChange(
	db *sql.DB,
	path string,
	cfg *config.Config,
	prevContent map[string]string,
	mu *sync.Mutex,
	pathStates map[string]*pathState,
	eventCh chan<- watchEvent,
	cwd string,
) {
	content, err := os.ReadFile(path)
	if err != nil {
		return
	}
	current := string(content)

	mu.Lock()
	prev := prevContent[path]
	prevContent[path] = current
	mu.Unlock()

	// Update dead-time tracking; reset if previously alerted so the timer restarts
	if state, exists := pathStates[path]; !exists || state.alerted {
		pathStates[path] = &pathState{firstWrite: time.Now(), writeCount: 1, alerted: false}
	} else {
		state.writeCount++
	}

	relPath, _ := filepath.Rel(cwd, path)

	// File-wide complexity — store always, display when above threshold
	score := analysis.Score(current)
	store.InsertComplexity(db, path, score)
	if score >= cfg.Thresholds.Complexity {
		avg, _ := store.AvgComplexity(db, path)
		avgPart := ""
		if avg > 0 {
			avgPart = fmt.Sprintf("  (your avg: %.0f)", avg)
		}
		fmt.Printf("grít  %s · complexity %.0f%s\n", pathStyle.Render(relPath), score, avgPart)

		// Function-level breakdown
		for _, fs := range analysis.ScoreByFunction(current) {
			if fs.Score >= cfg.Thresholds.Complexity {
				fmt.Printf("       └ func %s  complexity %.0f  (line %d)\n",
					pathStyle.Render(fs.Name), fs.Score, fs.Line)
			}
		}
	}

	// New-line set for naming and delta checks
	newLines := analysis.DiffLines(prev, current)
	prevLineCount := len(strings.Split(prev, "\n"))
	currLineCount := len(strings.Split(current, "\n"))
	netDeletion := prevLineCount - currLineCount

	// Language-aware naming check
	ext := filepath.Ext(path)
	langNames := cfg.Watch.LanguageNames[langForExt(ext)]
	if weakName := analysis.FindWeakNameWithExtra(newLines, langNames); weakName != "" {
		q := fmt.Sprintf("You used '%s' — what does it actually represent?", weakName)
		r := prompt.Ask(q, 30)
		if !r.Skipped && r.Answer != "" {
			id, _ := store.InsertEvent(db, "naming", false, "")
			store.InsertAnswer(db, id, q, r.Answer, "")
		} else {
			store.InsertEvent(db, "naming", true, "")
		}
	}

	// Paste vs AI-reflection (paste threshold is higher)
	if len(newLines) >= cfg.Thresholds.PasteLines {
		go func(p string, delta int) {
			time.Sleep(2 * time.Second)
			eventCh <- watchEvent{Type: evtPaste, Path: p, LinesDelta: delta}
		}(path, len(newLines))
	} else if len(newLines) >= cfg.Thresholds.AIReflectLines {
		go func(p string, delta int) {
			time.Sleep(2 * time.Second)
			eventCh <- watchEvent{Type: evtAIReflect, Path: p, LinesDelta: delta}
		}(path, len(newLines))
	}

	// Undo spike detection
	if prev != "" && netDeletion >= cfg.Thresholds.UndoSpikeLines {
		go func(p string, delta int) {
			time.Sleep(2 * time.Second)
			eventCh <- watchEvent{Type: evtUndoSpike, Path: p, LinesDelta: delta}
		}(path, netDeletion)
	}
}

func langForExt(ext string) string {
	switch ext {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".rs":
		return "rust"
	default:
		return ""
	}
}
