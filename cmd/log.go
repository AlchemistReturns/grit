package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/alchemistreturns/grit/internal/config"
	"github.com/alchemistreturns/grit/internal/store"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	logHook    string
	logSince   string
	logSkipped bool
)

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Show the friction timeline",
	RunE:  runLog,
}

func init() {
	logCmd.Flags().StringVar(&logHook, "hook", "", "filter by hook type (interview, file_complexity, naming, ai_reflect, decision, revert)")
	logCmd.Flags().StringVar(&logSince, "since", "", "show events since date (e.g. 2006-01-02)")
	logCmd.Flags().BoolVar(&logSkipped, "skipped", false, "show only skipped events")
	rootCmd.AddCommand(logCmd)
}

var (
	headerStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	metaStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	barStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	answerStyle2  = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	skipStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Italic(true)
	hookStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("75"))
	tagStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	revertStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	decisionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("141")).Bold(true)
)

func runLog(cmd *cobra.Command, args []string) error {
	if err := config.EnsureGritDir(); err != nil {
		return err
	}

	db, err := store.Open(config.DBPath())
	if err != nil {
		return err
	}
	defer db.Close()

	f := store.Filter{Hook: logHook}
	if logSince != "" {
		t, err := time.Parse("2006-01-02", logSince)
		if err != nil {
			return fmt.Errorf("invalid --since date: %w", err)
		}
		f.Since = t
	}
	if logSkipped {
		b := true
		f.Skipped = &b
	}

	events, err := store.QueryEvents(db, f)
	if err != nil {
		return err
	}

	if len(events) == 0 {
		fmt.Println(metaStyle.Render("no friction events recorded yet"))
		return nil
	}

	var lastDay string
	for _, ev := range events {
		day := ev.OccurredAt.Format("Monday, January 2 2006")
		if day != lastDay {
			fmt.Println()
			fmt.Println(headerStyle.Render("  " + day))
			lastDay = day
		}

		timeStr := ev.OccurredAt.Format("15:04")
		skipLabel := ""
		if ev.Skipped {
			skipLabel = "  " + skipStyle.Render("skipped")
		}

		commitPart := ""
		if ev.CommitMsg != "" {
			short := ev.CommitMsg
			if len(short) > 60 {
				short = short[:57] + "..."
			}
			commitPart = "  " + metaStyle.Render(short)
		}

		hookLabel := hookStyle.Render(ev.Hook)
		if ev.Hook == "revert" {
			hookLabel = revertStyle.Render("revert")
		} else if ev.Hook == "decision" {
			hookLabel = decisionStyle.Render("decision")
		}

		fmt.Printf("  %s  %s%s%s\n",
			metaStyle.Render(timeStr),
			hookLabel,
			commitPart,
			skipLabel,
		)

		if ev.RelatedCommit != "" {
			fmt.Printf("  %s reverts %s\n", barStyle.Render("  ┆"), metaStyle.Render(ev.RelatedCommit))
		}

		if !ev.Skipped {
			answers, err := store.QueryAnswersForEvent(db, ev.ID)
			if err == nil {
				if ev.Hook == "decision" {
					renderDecisionAnswers(answers)
				} else {
					renderAnswers(answers)
				}
			}
		}
	}
	fmt.Println()
	return nil
}

func renderAnswers(answers []store.Answer) {
	for i, a := range answers {
		prefix := barStyle.Render("  ┆ ")
		if i == len(answers)-1 {
			prefix = barStyle.Render("  └ ")
		}
		tagStr := ""
		if a.Tag != "" {
			tagStr = tagStyle.Render("[" + a.Tag + "] ")
		}
		fmt.Printf("%s%s\n", prefix, metaStyle.Render(a.Question))
		fmt.Printf("  %s %s%s\n", barStyle.Render("  "), tagStr, answerStyle2.Render(a.Answer))
	}
}

func renderDecisionAnswers(answers []store.Answer) {
	labels := map[string]string{
		"What situation or constraint is forcing this decision?":         "Context",
		"What alternatives did you evaluate? (separate with semicolons)": "Options considered",
		"What did you decide, and what was the deciding factor?":         "Decision",
		"What do you give up with this choice? What could go wrong?":     "Consequences",
	}
	for i, a := range answers {
		label := labels[a.Question]
		if label == "" {
			label = a.Question
		}
		prefix := barStyle.Render("  ┆ ")
		if i == len(answers)-1 {
			prefix = barStyle.Render("  └ ")
		}
		fmt.Printf("%s%s\n", prefix, metaStyle.Render(label))
		if a.Question == "What alternatives did you evaluate? (separate with semicolons)" {
			for j, opt := range strings.Split(a.Answer, ";") {
				opt = strings.TrimSpace(opt)
				if opt != "" {
					fmt.Printf("  %s %d. %s\n", barStyle.Render("  "), j+1, answerStyle2.Render(opt))
				}
			}
		} else {
			fmt.Printf("  %s %s\n", barStyle.Render("  "), answerStyle2.Render(a.Answer))
		}
	}
}
