package cmd

import (
	"os/exec"
	"strings"

	"github.com/alchemistreturns/grit/internal/config"
	"github.com/alchemistreturns/grit/internal/store"
	"github.com/spf13/cobra"
)

var postCommitCmd = &cobra.Command{
	Use:    "post-commit",
	Short:  "Record the commit hash for the latest interview (called by git post-commit hook)",
	Hidden: true,
	Run:    runPostCommit,
}

func init() {
	rootCmd.AddCommand(postCommitCmd)
}

func runPostCommit(cmd *cobra.Command, args []string) {
	if err := config.EnsureGritDir(); err != nil {
		return
	}

	db, err := store.Open(config.DBPath())
	if err != nil {
		return
	}
	defer db.Close()

	out, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		return
	}
	commitHash := strings.TrimSpace(string(out))
	if commitHash == "" {
		return
	}

	_ = store.UpdateLatestEventCommitHash(db, "interview", commitHash)
}
