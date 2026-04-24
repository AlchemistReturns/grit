package cmd

import (
	"fmt"
	"os"

	"github.com/alchemistreturns/grit/internal/config"
	"github.com/alchemistreturns/grit/internal/hooks"
	"github.com/alchemistreturns/grit/internal/store"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize grít in the current git repository",
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

const defaultGritYAML = `# grít configuration

questions:
  pool:
    - "What's the hardest part of this change?"
    - "What would you do differently next time?"
    - "What assumption are you making that could be wrong?"
    - "Did you have to look anything up? What was the gap?"
    - "Is this change reversible? If not, why not?"
    - "Who else needs to know about this change?"
    - "What did you cut from the original plan, and was that the right call?"
    - "What could break that you haven't tested?"
    - "What's the simplest thing you could have done here?"
    - "What would you tell a new engineer about this code tomorrow?"
    - "What slowed you down the most on this change?"
    - "What's the riskiest part of this commit?"
  window: 5

thresholds:
  complexity: 10.0
  ai_reflect_lines: 15
  dead_time_minutes: 40
  undo_spike_lines: 20
  paste_lines: 30

watch:
  extensions:
    - .go
    - .js
    - .ts
    - .py
    - .rs
  language_names:
    go: ["result", "tmp", "data", "obj"]
    python: ["data", "stuff", "res", "val"]

export:
  path: "~/.grit/exports"
  auto_export: false

deep_reflect:
  enabled: true
  output_dir: "~/.grit/reflections"
`

func runInit(cmd *cobra.Command, args []string) error {
	if err := config.EnsureGritDir(); err != nil {
		return fmt.Errorf("creating grit dir: %w", err)
	}

	db, err := store.Open(config.DBPath())
	if err != nil {
		return fmt.Errorf("opening store: %w", err)
	}
	db.Close()

	if _, err := os.Stat(".grit.yaml"); os.IsNotExist(err) {
		if err := os.WriteFile(".grit.yaml", []byte(defaultGritYAML), 0644); err != nil {
			return fmt.Errorf("writing .grit.yaml: %w", err)
		}
		fmt.Println("  created  .grit.yaml")
	} else {
		fmt.Println("  exists   .grit.yaml")
	}

	fmt.Printf("  created  %s\n", config.DBPath())

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		fmt.Println("  warning  no .git directory found — skipping hook install")
		return nil
	}
	if err := hooks.Install(cwd); err != nil {
		return fmt.Errorf("installing pre-commit hook: %w", err)
	}
	fmt.Println("  created  .git/hooks/pre-commit")

	if err := hooks.InstallPostRewrite(cwd); err != nil {
		return fmt.Errorf("installing post-rewrite hook: %w", err)
	}
	fmt.Println("  created  .git/hooks/post-rewrite")

	fmt.Println("\ngrít initialized. Start committing to capture friction.")
	return nil
}
