package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alchemistreturns/grit/internal/config"
	"github.com/alchemistreturns/grit/internal/store"
	"github.com/spf13/cobra"
)

var (
	pushMD    bool
	pushJSON  bool
	pushSince string
)

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Export friction data to Markdown or JSON",
	RunE:  runPush,
}

func init() {
	pushCmd.Flags().BoolVar(&pushMD, "md", false, "export to Markdown")
	pushCmd.Flags().BoolVar(&pushJSON, "json", false, "export to JSON")
	pushCmd.Flags().StringVar(&pushSince, "since", "", "export since date (YYYY-MM-DD, default: current month)")
	rootCmd.AddCommand(pushCmd)
}

type exportAnswer struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
	Tag      string `json:"tag,omitempty"`
}

type exportEvent struct {
	ID         string         `json:"id"`
	Hook       string         `json:"hook"`
	OccurredAt string         `json:"occurred_at"`
	CommitMsg  string         `json:"commit_msg,omitempty"`
	Answers    []exportAnswer `json:"answers,omitempty"`
}

type exportRoot struct {
	ExportedAt string        `json:"exported_at"`
	Events     []exportEvent `json:"events"`
}

func runPush(cmd *cobra.Command, args []string) error {
	if !pushMD && !pushJSON {
		return fmt.Errorf("specify --md or --json")
	}

	if err := config.EnsureGritDir(); err != nil {
		return err
	}
	db, err := store.Open(config.DBPath())
	if err != nil {
		return err
	}
	defer db.Close()

	since := time.Now().AddDate(0, -1, 0) // default: last month
	if pushSince != "" {
		t, err := time.Parse("2006-01-02", pushSince)
		if err != nil {
			return fmt.Errorf("invalid --since date: %w", err)
		}
		since = t
	}

	events, err := store.QueryEvents(db, store.Filter{Since: since})
	if err != nil {
		return err
	}

	// Build export payload
	var payload []exportEvent
	for _, ev := range events {
		if ev.Skipped {
			continue
		}
		answers, _ := store.QueryAnswersForEvent(db, ev.ID)
		var expAnswers []exportAnswer
		for _, a := range answers {
			expAnswers = append(expAnswers, exportAnswer{
				Question: a.Question,
				Answer:   a.Answer,
				Tag:      a.Tag,
			})
		}
		payload = append(payload, exportEvent{
			ID:         ev.ID,
			Hook:       ev.Hook,
			OccurredAt: ev.OccurredAt.UTC().Format(time.RFC3339),
			CommitMsg:  ev.CommitMsg,
			Answers:    expAnswers,
		})
	}

	cfg, _ := config.Load()
	exportPath := config.ExpandPath(cfg.Export.Path)
	if err := os.MkdirAll(exportPath, 0755); err != nil {
		return fmt.Errorf("creating export dir: %w", err)
	}

	label := since.Format("2006-01")
	if pushSince != "" {
		label = since.Format("2006-01-02")
	}

	if pushJSON {
		root := exportRoot{
			ExportedAt: time.Now().UTC().Format(time.RFC3339),
			Events:     payload,
		}
		data, err := json.MarshalIndent(root, "", "  ")
		if err != nil {
			return err
		}
		outFile := filepath.Join(exportPath, fmt.Sprintf("grit-friction-%s.json", label))
		if err := os.WriteFile(outFile, data, 0644); err != nil {
			return err
		}
		fmt.Printf("  wrote  %s  (%d events)\n", outFile, len(payload))
	}

	if pushMD {
		outFile := filepath.Join(exportPath, fmt.Sprintf("grit-friction-%s.md", label))
		md := buildMarkdownExport(label, payload)
		if err := os.WriteFile(outFile, []byte(md), 0644); err != nil {
			return err
		}
		fmt.Printf("  wrote  %s  (%d events)\n", outFile, len(payload))
	}

	return nil
}

func buildMarkdownExport(label string, events []exportEvent) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# Grít Friction Log — %s\n\n", label)

	lastDay := ""
	for _, ev := range events {
		t, _ := time.Parse(time.RFC3339, ev.OccurredAt)
		day := t.Format("Monday, January 2 2006")
		if day != lastDay {
			fmt.Fprintf(&sb, "\n## %s\n\n", day)
			lastDay = day
		}

		timeStr := t.Format("15:04")
		commitPart := ""
		if ev.CommitMsg != "" {
			short := ev.CommitMsg
			if len(short) > 60 {
				short = short[:57] + "..."
			}
			commitPart = "  —  " + short
		}
		fmt.Fprintf(&sb, "### %s  `%s`%s\n\n", timeStr, ev.Hook, commitPart)

		for _, a := range ev.Answers {
			tagStr := ""
			if a.Tag != "" {
				tagStr = fmt.Sprintf(" `[%s]`", a.Tag)
			}
			fmt.Fprintf(&sb, "**%s**%s\n\n> %s\n\n", a.Question, tagStr, a.Answer)
		}
	}

	return sb.String()
}
