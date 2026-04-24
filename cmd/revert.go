package cmd

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	"github.com/alchemistreturns/grit/internal/config"
	"github.com/alchemistreturns/grit/internal/prompt"
	"github.com/alchemistreturns/grit/internal/store"
	"github.com/spf13/cobra"
)

var revertCheckFlag bool

var revertCmd = &cobra.Command{
	Use:   "revert",
	Short: "Record a post-mortem for a reverted commit",
	RunE:  runRevert,
}

func init() {
	revertCmd.Flags().BoolVar(&revertCheckFlag, "check", false, "check if latest commit is a revert (used by post-rewrite hook)")
	rootCmd.AddCommand(revertCmd)
}

var hashRe = regexp.MustCompile(`[0-9a-f]{7,40}`)

func runRevert(cmd *cobra.Command, args []string) error {
	if err := config.EnsureGritDir(); err != nil {
		return err
	}
	db, err := store.Open(config.DBPath())
	if err != nil {
		return err
	}
	defer db.Close()

	if revertCheckFlag {
		// Called from post-rewrite hook — check COMMIT_EDITMSG silently
		msg := readCommitMsg()
		if !isRevert(msg) {
			return nil
		}
		if !hasTTY() {
			store.InsertEvent(db, "revert", true, msg)
			return nil
		}
		relatedHash := extractRevertHash(msg)
		runRevertInterview(db, msg, relatedHash)
		return nil
	}

	// Manual invocation — prompt for original commit message
	if !hasTTY() {
		fmt.Println("grit revert: no TTY available")
		return nil
	}
	runRevertInterview(db, "", "")
	return nil
}

// runRevertInterview runs a 3-question post-mortem and stores a revert event.
// Called by both cmd/revert.go and cmd/commit.go (pre-commit revert detection).
func runRevertInterview(db *sql.DB, commitMsg, relatedCommit string) {
	originalMsg := extractRevertOriginal(commitMsg)
	questionText := "What went wrong with the original change?"
	if originalMsg != "" {
		questionText = fmt.Sprintf("What went wrong with \"%s\"?", truncate(originalMsg, 60))
	}

	questions := []prompt.Question{
		{Text: questionText, TimeoutSec: 30},
		{Text: "Was this caught in review, or did it reach production?", TimeoutSec: 30},
		{Text: "What would have caught this earlier?", TimeoutSec: 30},
	}

	result := prompt.Run(questions)

	allSkipped := true
	for _, r := range result.Answers {
		if !r.Skipped && r.Answer != "" {
			allSkipped = false
			break
		}
	}

	eventID, err := store.InsertEventFull(db, "revert", allSkipped, commitMsg, relatedCommit)
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

// extractRevertOriginal parses the original commit message from a revert commit message.
// Format: Revert "original commit message"
func extractRevertOriginal(msg string) string {
	if !strings.HasPrefix(strings.ToLower(msg), "revert ") {
		return ""
	}
	msg = strings.TrimSpace(msg)
	if start := strings.Index(msg, `"`); start >= 0 {
		rest := msg[start+1:]
		if end := strings.LastIndex(rest, `"`); end >= 0 {
			return rest[:end]
		}
	}
	return ""
}

// extractRevertHash parses the reverted commit hash from the revert body.
// Format: This reverts commit <hash>.
func extractRevertHash(msg string) string {
	lower := strings.ToLower(msg)
	if idx := strings.Index(lower, "this reverts commit "); idx >= 0 {
		rest := msg[idx+len("this reverts commit "):]
		if m := hashRe.FindString(rest); m != "" {
			return m
		}
	}
	return ""
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
