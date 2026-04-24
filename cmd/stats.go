package cmd

import (
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/alchemistreturns/grit/internal/config"
	"github.com/alchemistreturns/grit/internal/store"
	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show friction analytics",
}

var statsWeekCmd = &cobra.Command{
	Use:   "week",
	Short: "Show the past 7 days of friction data",
	RunE:  runStatsWeek,
}

var statsFileCmd = &cobra.Command{
	Use:   "file <path>",
	Short: "Show one file's complexity trend and friction notes",
	Args:  cobra.ExactArgs(1),
	RunE:  runStatsFile,
}

var statsHeatmapCmd = &cobra.Command{
	Use:   "heatmap",
	Short: "Render a contribution-style heatmap of friction density",
	RunE:  runStatsHeatmap,
}

var statsDigestCmd = &cobra.Command{
	Use:   "digest",
	Short: "Print a weekly digest grouped by tag",
	RunE:  runStatsDigest,
}

func init() {
	statsCmd.AddCommand(statsWeekCmd)
	statsCmd.AddCommand(statsFileCmd)
	statsCmd.AddCommand(statsHeatmapCmd)
	statsCmd.AddCommand(statsDigestCmd)
	rootCmd.AddCommand(statsCmd)
}

func openDB() (*sql.DB, error) {
	if err := config.EnsureGritDir(); err != nil {
		return nil, err
	}
	return store.Open(config.DBPath())
}

func runStatsWeek(cmd *cobra.Command, args []string) error {
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	since := time.Now().AddDate(0, 0, -7).Unix()
	total, skipped, err := store.CountEvents(db, since)
	if err != nil {
		return err
	}
	completed := total - skipped
	streak, _ := store.StreakDays(db)
	tagCounts, _ := store.TagCounts(db, since)

	fmt.Println()
	fmt.Println(headerStyle.Render("  Past 7 days"))
	fmt.Println()
	fmt.Printf("  Commits:   %d total  ·  %d completed  ·  %d skipped\n", total, completed, skipped)
	fmt.Printf("  Streak:    %d consecutive days with a completed interview\n", streak)

	if len(tagCounts) > 0 {
		fmt.Println()
		fmt.Println(metaStyle.Render("  Top friction tags:"))
		for tag, count := range tagCounts {
			bar := strings.Repeat("█", min(count, 20))
			fmt.Printf("    [%-12s]  %s  %d\n", tag, bar, count)
		}
	}

	rows, err := db.Query(`
		SELECT path, MAX(score) as max_score
		FROM complexity_history
		WHERE recorded_at >= ?
		GROUP BY path
		ORDER BY max_score DESC
		LIMIT 5
	`, since)
	if err == nil {
		defer rows.Close()
		first := true
		for rows.Next() {
			var path string
			var score float64
			if err := rows.Scan(&path, &score); err != nil {
				continue
			}
			if first {
				fmt.Println()
				fmt.Println(metaStyle.Render("  Most complex files touched:"))
				first = false
			}
			fmt.Printf("    %s  (peak complexity %.0f)\n", pathStyle.Render(path), score)
		}
	}
	fmt.Println()
	return nil
}

func runStatsFile(cmd *cobra.Command, args []string) error {
	path := args[0]
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	fmt.Println()
	fmt.Println(headerStyle.Render("  " + path))
	fmt.Println()

	scores, _ := store.ComplexityHistory(db, path, 20)
	if len(scores) > 0 {
		fmt.Printf("  Complexity trend:  %s\n", sparkline(scores))
	}

	answers, _ := store.AnswersForPath(db, path)
	if len(answers) > 0 {
		fmt.Println()
		fmt.Println(metaStyle.Render("  Friction notes:"))
		for _, a := range answers {
			tagStr := ""
			if a.Tag != "" {
				tagStr = tagStyle.Render("["+a.Tag+"]") + " "
			}
			fmt.Printf("    %s%s\n", tagStr, a.Answer)
		}
	} else {
		fmt.Println(metaStyle.Render("  No friction notes recorded for this path."))
	}
	fmt.Println()
	return nil
}

func runStatsHeatmap(cmd *cobra.Command, args []string) error {
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	weeks := 12
	now := time.Now()

	// Monday of the current week
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday
	}
	weekStart := time.Date(now.Year(), now.Month(), now.Day()-(weekday-1), 0, 0, 0, 0, now.Location())

	since := weekStart.AddDate(0, 0, -(weeks-1)*7)
	eventsPerDay, err := store.EventsPerDay(db, since.Unix())
	if err != nil {
		return err
	}

	dayNames := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}

	fmt.Println()
	for dow := 0; dow < 7; dow++ {
		fmt.Printf("  %s ", dayNames[dow])
		for w := 0; w < weeks; w++ {
			day := weekStart.AddDate(0, 0, -(weeks-1-w)*7+dow)
			if day.After(now) {
				fmt.Print(" ")
				continue
			}
			key := day.Format("2006-01-02")
			fmt.Print(densityChar(eventsPerDay[key]))
		}
		fmt.Println()
	}

	// Month labels
	fmt.Print("       ")
	lastMonth := ""
	for w := 0; w < weeks; w++ {
		day := weekStart.AddDate(0, 0, -(weeks-1-w)*7)
		month := day.Format("Jan")
		if month != lastMonth && day.Day() <= 7 {
			fmt.Printf("%-4s", month)
			lastMonth = month
		} else {
			fmt.Print("    ")
		}
	}
	fmt.Println()
	fmt.Println()
	fmt.Printf("  %s░ 0  ▒ 1-2  ▓ 3-4  █ 5+%s\n", metaStyle.Render(""), metaStyle.Render(""))
	fmt.Println()
	return nil
}

func densityChar(count int) string {
	switch {
	case count == 0:
		return "░"
	case count <= 2:
		return "▒"
	case count <= 4:
		return "▓"
	default:
		return "█"
	}
}

func runStatsDigest(cmd *cobra.Command, args []string) error {
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	since := time.Now().AddDate(0, 0, -7)
	events, err := store.QueryEvents(db, store.Filter{Since: since})
	if err != nil {
		return err
	}

	byTag := make(map[string][]string)
	var untagged []string

	for _, ev := range events {
		if ev.Skipped {
			continue
		}
		answers, _ := store.QueryAnswersForEvent(db, ev.ID)
		for _, a := range answers {
			if a.Tag != "" {
				byTag[a.Tag] = append(byTag[a.Tag], a.Answer)
			} else {
				untagged = append(untagged, a.Answer)
			}
		}
	}

	fmt.Println()
	fmt.Println(headerStyle.Render("  Weekly Digest"))
	fmt.Println()

	if len(byTag) == 0 && len(untagged) == 0 {
		fmt.Println(metaStyle.Render("  No answered interviews this week."))
		fmt.Println()
		return nil
	}

	for tag, answers := range byTag {
		fmt.Printf("  [%s]\n", tag)
		for _, a := range answers {
			fmt.Printf("    • %s\n", a)
		}
		fmt.Println()
	}

	if len(untagged) > 0 {
		fmt.Println("  [general]")
		for _, a := range untagged {
			fmt.Printf("    • %s\n", a)
		}
		fmt.Println()
	}

	return nil
}

func sparkline(scores []float64) string {
	if len(scores) == 0 {
		return ""
	}
	// Scores come newest-first; reverse for left-to-right display
	reversed := make([]float64, len(scores))
	for i, s := range scores {
		reversed[len(scores)-1-i] = s
	}

	blocks := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	minS, maxS := reversed[0], reversed[0]
	for _, s := range reversed {
		if s < minS {
			minS = s
		}
		if s > maxS {
			maxS = s
		}
	}

	var sb strings.Builder
	for _, s := range reversed {
		idx := 0
		if maxS > minS {
			idx = int(math.Round((s - minS) / (maxS - minS) * float64(len(blocks)-1)))
		}
		sb.WriteRune(blocks[idx])
	}
	return sb.String()
}
