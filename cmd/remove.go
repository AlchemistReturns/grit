package cmd

import (
	"fmt"
	"os"

	"github.com/alchemistreturns/grit/internal/config"
	"github.com/alchemistreturns/grit/internal/hooks"
	"github.com/spf13/cobra"
)

var removeAll bool

var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove grít hooks and configuration",
	RunE:  runRemove,
}

func init() {
	removeCmd.Flags().BoolVar(&removeAll, "all", false, "completely remove the .grit folder including the database")
	rootCmd.AddCommand(removeCmd)
}

func runRemove(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	if _, err := os.Stat(".git"); err == nil {
		if err := hooks.Uninstall(cwd); err != nil {
			fmt.Printf("  warning  failed to uninstall pre-commit hook: %v\n", err)
		} else {
			fmt.Println("  removed  .git/hooks/pre-commit (grít logic)")
		}

		if err := hooks.UninstallPostRewrite(cwd); err != nil {
			fmt.Printf("  warning  failed to uninstall post-rewrite hook: %v\n", err)
		} else {
			fmt.Println("  removed  .git/hooks/post-rewrite (grít logic)")
		}

		if err := hooks.UninstallPostCommit(cwd); err != nil {
			fmt.Printf("  warning  failed to uninstall post-commit hook: %v\n", err)
		} else {
			fmt.Println("  removed  .git/hooks/post-commit (grít logic)")
		}
	}

	if _, err := os.Stat(".grit.yaml"); err == nil {
		if err := os.Remove(".grit.yaml"); err != nil {
			fmt.Printf("  warning  failed to remove .grit.yaml: %v\n", err)
		} else {
			fmt.Println("  removed  .grit.yaml")
		}
	}

	if removeAll {
		gritDir := config.GritDir()
		if _, err := os.Stat(gritDir); err == nil {
			if err := os.RemoveAll(gritDir); err != nil {
				return fmt.Errorf("failed to remove .grit directory: %w", err)
			}
			fmt.Println("  removed  .grit directory")
		}
	} else {
		fmt.Println("\nTip: Use 'grit remove --all' to completely remove the .grit folder and database.")
	}

	fmt.Println("\ngrít has been removed from this repository.")
	return nil
}
