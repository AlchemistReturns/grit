package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/alchemistreturns/grit/internal/config"
	"github.com/alchemistreturns/grit/internal/prompt"
	"github.com/alchemistreturns/grit/internal/store"
	"github.com/spf13/cobra"
)

var decisionCmd = &cobra.Command{
	Use:   "decision",
	Short: "Record an architectural decision",
	RunE:  runDecision,
}

var decisionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all recorded decisions",
	RunE:  runDecisionList,
}

var decisionExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export decisions as ADR markdown files",
	RunE:  runDecisionExport,
}

func init() {
	decisionCmd.AddCommand(decisionListCmd)
	decisionCmd.AddCommand(decisionExportCmd)
	rootCmd.AddCommand(decisionCmd)
}

var decisionQuestions = []prompt.Question{
	{Text: "What situation or constraint is forcing this decision?", TimeoutSec: 60},
	{Text: "What alternatives did you evaluate? (separate with semicolons)", TimeoutSec: 60},
	{Text: "What did you decide, and what was the deciding factor?", TimeoutSec: 60},
	{Text: "What do you give up with this choice? What could go wrong?", TimeoutSec: 60},
}

func runDecision(cmd *cobra.Command, args []string) error {
	if err := config.EnsureGritDir(); err != nil {
		return err
	}
	db, err := store.Open(config.DBPath())
	if err != nil {
		return err
	}
	defer db.Close()

	if !hasTTY() {
		return fmt.Errorf("grit decision: requires an interactive terminal")
	}

	fmt.Println()
	fmt.Println(decisionStyle.Render("  Architectural Decision Record"))
	fmt.Println(metaStyle.Render("  Answer each question — press Esc to skip, Enter to submit."))
	fmt.Println()

	result := prompt.Run(decisionQuestions)

	allSkipped := true
	for _, r := range result.Answers {
		if !r.Skipped && r.Answer != "" {
			allSkipped = false
			break
		}
	}

	eventID, err := store.InsertEvent(db, "decision", allSkipped, "")
	if err != nil {
		return err
	}

	for i, r := range result.Answers {
		if !r.Skipped && r.Answer != "" {
			store.InsertAnswer(db, eventID, decisionQuestions[i].Text, r.Answer, "")
		}
	}

	if !allSkipped {
		fmt.Println(metaStyle.Render("\n  Decision recorded. Use `grit log --hook decision` to review."))
	}
	return nil
}

func runDecisionList(cmd *cobra.Command, args []string) error {
	if err := config.EnsureGritDir(); err != nil {
		return err
	}
	db, err := store.Open(config.DBPath())
	if err != nil {
		return err
	}
	defer db.Close()

	events, err := store.QueryEvents(db, store.Filter{Hook: "decision"})
	if err != nil {
		return err
	}

	if len(events) == 0 {
		fmt.Println(metaStyle.Render("No decisions recorded yet. Run `grit decision` to record one."))
		return nil
	}

	fmt.Println()
	for _, ev := range events {
		answers, _ := store.QueryAnswersForEvent(db, ev.ID)
		title := decisionTitle(answers)
		fmt.Printf("  %s  %s  %s\n",
			metaStyle.Render(ev.OccurredAt.Format("2006-01-02")),
			decisionStyle.Render("decision"),
			answerStyle2.Render(title),
		)
	}
	fmt.Println()
	return nil
}

func runDecisionExport(cmd *cobra.Command, args []string) error {
	if err := config.EnsureGritDir(); err != nil {
		return err
	}
	db, err := store.Open(config.DBPath())
	if err != nil {
		return err
	}
	defer db.Close()

	events, err := store.QueryEvents(db, store.Filter{Hook: "decision"})
	if err != nil {
		return err
	}

	if len(events) == 0 {
		fmt.Println(metaStyle.Render("No decisions to export."))
		return nil
	}

	if err := os.MkdirAll("decisions", 0755); err != nil {
		return fmt.Errorf("creating decisions/ dir: %w", err)
	}

	count := 0
	for _, ev := range events {
		if ev.Skipped {
			continue
		}
		answers, _ := store.QueryAnswersForEvent(db, ev.ID)
		md := formatDecisionADR(ev.OccurredAt, answers)
		title := decisionTitle(answers)
		slug := toSlug(title)
		filename := fmt.Sprintf("decisions/%s-%s.md", ev.OccurredAt.Format("2006-01-02"), slug)
		if err := os.WriteFile(filename, []byte(md), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to write %s: %v\n", filename, err)
			continue
		}
		fmt.Printf("  wrote  %s\n", filepath.Clean(filename))
		count++
	}

	fmt.Printf("\n  %d decision(s) exported to decisions/\n", count)
	return nil
}

func decisionTitle(answers []store.Answer) string {
	for _, a := range answers {
		if strings.Contains(a.Question, "What did you decide") {
			if len(a.Answer) > 60 {
				return a.Answer[:57] + "..."
			}
			return a.Answer
		}
	}
	return "decision"
}

func formatDecisionADR(date time.Time, answers []store.Answer) string {
	answerMap := make(map[string]string)
	for _, a := range answers {
		answerMap[a.Question] = a.Answer
	}

	context := answerMap["What situation or constraint is forcing this decision?"]
	options := answerMap["What alternatives did you evaluate? (separate with semicolons)"]
	decision := answerMap["What did you decide, and what was the deciding factor?"]
	consequences := answerMap["What do you give up with this choice? What could go wrong?"]

	title := decision
	if title == "" {
		title = "Decision"
	}
	if len(title) > 60 {
		title = title[:57] + "..."
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "# Decision: %s\n\n", title)
	fmt.Fprintf(&sb, "Date: %s\n\n", date.Format("2006-01-02"))

	if context != "" {
		fmt.Fprintf(&sb, "## Context\n%s\n\n", context)
	}
	if options != "" {
		fmt.Fprintf(&sb, "## Options Considered\n")
		for i, opt := range strings.Split(options, ";") {
			opt = strings.TrimSpace(opt)
			if opt != "" {
				fmt.Fprintf(&sb, "%d. %s\n", i+1, opt)
			}
		}
		fmt.Fprintln(&sb)
	}
	if decision != "" {
		fmt.Fprintf(&sb, "## Decision\n%s\n\n", decision)
	}
	if consequences != "" {
		fmt.Fprintf(&sb, "## Consequences\n%s\n", consequences)
	}

	return sb.String()
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func toSlug(s string) string {
	s = strings.ToLower(s)
	s = slugRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 50 {
		s = s[:50]
	}
	return s
}
