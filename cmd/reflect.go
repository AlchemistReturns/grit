package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alchemistreturns/grit/internal/config"
	"github.com/alchemistreturns/grit/internal/prompt"
	"github.com/alchemistreturns/grit/internal/store"
	"github.com/spf13/cobra"
)

var reflectCmd = &cobra.Command{
	Use:   "reflect",
	Short: "End-of-day review",
	RunE:  runReflect,
}

func init() {
	rootCmd.AddCommand(reflectCmd)
}

var deepReflectPool = []string{
	"What problem did you think you were solving this morning versus what you actually solved?",
	"What would you tell a teammate who picks up this code tomorrow?",
	"What would you have done differently if you had one more hour?",
	"What's the most surprising thing you learned today?",
	"What decision today might you regret in 6 months?",
	"Where did your estimate go wrong today, and why?",
}

func runReflect(cmd *cobra.Command, args []string) error {
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

	if !hasTTY() {
		return fmt.Errorf("grit reflect: requires an interactive terminal")
	}

	today := time.Now().Truncate(24 * time.Hour)
	total, skipped, _ := store.CountEvents(db, today.Unix())
	completed := total - skipped

	fmt.Println()
	fmt.Println(headerStyle.Render("  End-of-Day Reflection"))
	fmt.Println()
	fmt.Printf("  Today: %d friction events  ·  %d interviews completed  ·  %d skipped\n",
		total, completed, skipped)
	fmt.Println()

	// Pick 1-3 questions, avoiding recently used ones
	recent, _ := store.RecentQuestions(db, 3)
	recentSet := make(map[string]struct{})
	for _, q := range recent {
		recentSet[q] = struct{}{}
	}

	var questions []prompt.Question
	for _, q := range deepReflectPool {
		if _, seen := recentSet[q]; !seen {
			questions = append(questions, prompt.Question{Text: q, TimeoutSec: 60})
			if len(questions) >= 2 {
				break
			}
		}
	}
	if len(questions) == 0 {
		questions = []prompt.Question{{Text: deepReflectPool[0], TimeoutSec: 60}}
	}

	result := prompt.Run(questions)

	allSkipped := true
	for _, r := range result.Answers {
		if !r.Skipped && r.Answer != "" {
			allSkipped = false
			break
		}
	}

	eventID, err := store.InsertEvent(db, "reflect", allSkipped, "")
	if err != nil {
		return err
	}

	for i, r := range result.Answers {
		if !r.Skipped && r.Answer != "" {
			store.InsertAnswer(db, eventID, questions[i].Text, r.Answer, "")
		}
	}

	// Write to dated markdown file if deep_reflect is enabled
	if !allSkipped && cfg.DeepReflect.Enabled {
		if err := writeReflectionMarkdown(cfg, today, questions, result.Answers); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to save reflection: %v\n", err)
		} else {
			outDir := config.ExpandPath(cfg.DeepReflect.OutputDir)
			fmt.Printf("\n  %s\n", metaStyle.Render("Reflection saved to "+filepath.Join(outDir, today.Format("2006-01-02")+".md")))
		}
	}

	fmt.Println()
	return nil
}

func writeReflectionMarkdown(cfg *config.Config, date time.Time, questions []prompt.Question, answers []prompt.Result) error {
	outDir := config.ExpandPath(cfg.DeepReflect.OutputDir)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}
	filename := filepath.Join(outDir, date.Format("2006-01-02")+".md")

	var sb strings.Builder
	fmt.Fprintf(&sb, "# Reflection — %s\n\n", date.Format("Monday, January 2 2006"))

	for i, q := range questions {
		if i < len(answers) && !answers[i].Skipped && answers[i].Answer != "" {
			fmt.Fprintf(&sb, "## %s\n\n%s\n\n", q.Text, answers[i].Answer)
		}
	}

	return os.WriteFile(filename, []byte(sb.String()), 0644)
}
