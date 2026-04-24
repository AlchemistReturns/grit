package cmd

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/alchemistreturns/grit/internal/config"
	"github.com/alchemistreturns/grit/internal/prompt"
	"github.com/alchemistreturns/grit/internal/store"
	"github.com/spf13/cobra"
)

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Run the friction interview (called by git pre-commit hook)",
	Run:   runCommit,
}

func init() {
	rootCmd.AddCommand(commitCmd)
}

func runCommit(cmd *cobra.Command, args []string) {
	// Guarantee exit 0 — never block a commit
	defer func() {
		if r := recover(); r != nil {
			os.Exit(0)
		}
		os.Exit(0)
	}()

	if err := config.EnsureGritDir(); err != nil {
		return
	}

	cfg, err := config.Load()
	if err != nil {
		return
	}

	db, err := store.Open(config.DBPath())
	if err != nil {
		return
	}
	defer db.Close()

	commitMsg := readCommitMsg()

	// Detect reverts and run post-mortem instead
	if isRevert(commitMsg) {
		if hasTTY() {
			relatedHash := extractRevertHash(commitMsg)
			runRevertInterview(db, commitMsg, relatedHash)
		} else {
			store.InsertEvent(db, "revert", true, commitMsg)
		}
		return
	}

	if shouldSkip(commitMsg) {
		store.InsertEvent(db, "interview", true, commitMsg)
		return
	}

	if !hasTTY() {
		store.InsertEvent(db, "interview", true, commitMsg)
		return
	}

	questions := pickQuestions(db, cfg)
	result := prompt.Run(questions)

	allSkipped := true
	for _, r := range result.Answers {
		if !r.Skipped && r.Answer != "" {
			allSkipped = false
			break
		}
	}

	eventID, err := store.InsertEvent(db, "interview", allSkipped, commitMsg)
	if err != nil {
		return
	}

	for i, r := range result.Answers {
		if !r.Skipped && r.Answer != "" {
			tag, text := parseTag(r.Answer)
			store.InsertAnswer(db, eventID, questions[i].Text, text, tag)
		}
	}
}

// pickQuestions builds the interview question list using adaptive rotation and commit-type awareness.
func pickQuestions(db *sql.DB, cfg *config.Config) []prompt.Question {
	recent, _ := store.RecentQuestions(db, cfg.Questions.Window)
	recentSet := make(map[string]struct{}, len(recent))
	for _, q := range recent {
		recentSet[q] = struct{}{}
	}

	pool := cfg.Questions.Pool
	var fresh, used []string
	for _, q := range pool {
		if _, seen := recentSet[q]; seen {
			used = append(used, q)
		} else {
			fresh = append(fresh, q)
		}
	}
	available := append(fresh, used...) // fresh questions first, then cycle back

	var questions []prompt.Question

	// Prepend commit-type specific question if applicable
	if typeQ := commitTypeQuestion(); typeQ != nil {
		questions = append(questions, *typeQ)
	}

	for _, q := range available {
		if len(questions) >= 2 {
			break
		}
		questions = append(questions, prompt.Question{Text: q, TimeoutSec: 30})
	}

	if len(questions) == 0 && len(pool) > 0 {
		questions = []prompt.Question{{Text: pool[0], TimeoutSec: 30}}
	}
	return questions
}

// commitTypeQuestion returns a situational question based on the staged diff.
func commitTypeQuestion() *prompt.Question {
	numstatOut, err := exec.Command("git", "diff", "--cached", "--numstat").Output()
	if err != nil || len(strings.TrimSpace(string(numstatOut))) == 0 {
		return nil
	}

	totalAdded, totalRemoved := 0, 0
	allTest := true
	allConfig := true
	fileCount := 0

	for _, line := range strings.Split(strings.TrimSpace(string(numstatOut)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}
		fileCount++
		added, _ := strconv.Atoi(parts[0])
		removed, _ := strconv.Atoi(parts[1])
		name := parts[2]
		totalAdded += added
		totalRemoved += removed
		if !isTestFile(name) {
			allTest = false
		}
		if !isConfigFile(name) {
			allConfig = false
		}
	}

	hasNewFile := false
	nsOut, err := exec.Command("git", "diff", "--cached", "--name-status").Output()
	if err == nil {
		for _, line := range strings.Split(string(nsOut), "\n") {
			if strings.HasPrefix(line, "A\t") || strings.HasPrefix(line, "A ") {
				hasNewFile = true
				break
			}
		}
	}

	total := totalAdded + totalRemoved
	var q string
	switch {
	case total > 200:
		q = fmt.Sprintf("This diff touches %d lines — should it be split into smaller commits?", total)
	case hasNewFile:
		q = "You added a new file — what contract or responsibility does it own?"
	case fileCount > 0 && allTest:
		q = "This is a test-only commit — what edge case drove you to write this?"
	case fileCount > 0 && allConfig:
		q = "You changed config — what breaks if this value is wrong?"
	}

	if q == "" {
		return nil
	}
	return &prompt.Question{Text: q, TimeoutSec: 30}
}

// parseTag extracts an optional [tag] prefix from an answer string.
// "[debug] spent 2h on wrong assumption" -> ("debug", "spent 2h on wrong assumption")
func parseTag(answer string) (tag, text string) {
	if strings.HasPrefix(answer, "[") {
		if end := strings.Index(answer, "]"); end > 0 {
			return strings.TrimSpace(answer[1:end]), strings.TrimSpace(answer[end+1:])
		}
	}
	return "", answer
}

func isTestFile(path string) bool {
	p := strings.ToLower(path)
	return strings.Contains(p, "_test") || strings.Contains(p, "test_") ||
		strings.Contains(p, "/test/") || strings.Contains(p, "/tests/") ||
		strings.Contains(p, "spec")
}

func isConfigFile(path string) bool {
	p := strings.ToLower(path)
	for _, ext := range []string{".yaml", ".yml", ".json", ".toml", ".env", ".ini", ".conf"} {
		if strings.HasSuffix(p, ext) {
			return true
		}
	}
	return strings.Contains(p, "config") || strings.Contains(p, "setting")
}

func readCommitMsg() string {
	gitDir := os.Getenv("GIT_DIR")
	if gitDir == "" {
		gitDir = ".git"
	}
	data, err := os.ReadFile(gitDir + "/COMMIT_EDITMSG")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func shouldSkip(msg string) bool {
	lower := strings.ToLower(msg)
	prefixes := []string{"merge ", "fixup!", "squash!", "wip:", "amend"}
	for _, p := range prefixes {
		if strings.HasPrefix(lower, p) {
			return true
		}
	}
	return false
}

func isRevert(msg string) bool {
	return strings.HasPrefix(strings.ToLower(msg), "revert ")
}

func hasTTY() bool {
	// Check stdin first (direct invocation)
	if fi, err := os.Stdin.Stat(); err == nil && fi.Mode()&os.ModeCharDevice != 0 {
		return true
	}
	// Git hooks redirect stdin to /dev/null, but the console is still reachable.
	// Try the platform-specific terminal device that tea.WithInputTTY() also uses.
	for _, dev := range []string{"CONIN$", "/dev/tty"} {
		if f, err := os.Open(dev); err == nil {
			f.Close()
			return true
		}
	}
	return false
}
