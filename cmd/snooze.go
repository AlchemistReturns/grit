package cmd

import (
	"fmt"
	"time"

	"github.com/alchemistreturns/grit/internal/config"
	"github.com/spf13/cobra"
)

var snoozeCmd = &cobra.Command{
	Use:   "snooze [duration]",
	Short: "Pause friction interviews for a duration (default 1h)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runSnooze,
}

var resumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Re-enable friction interviews",
	RunE:  runResume,
}

var disableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable friction interviews indefinitely",
	RunE:  runDisable,
}

func init() {
	rootCmd.AddCommand(snoozeCmd)
	rootCmd.AddCommand(resumeCmd)
	rootCmd.AddCommand(disableCmd)
}

func runSnooze(cmd *cobra.Command, args []string) error {
	if err := config.EnsureGritDir(); err != nil {
		return err
	}
	d := time.Hour
	if len(args) == 1 {
		var err error
		d, err = time.ParseDuration(args[0])
		if err != nil {
			return fmt.Errorf("invalid duration %q — use Go duration format, e.g. 30m, 2h, 1h30m", args[0])
		}
	}
	if err := config.Pause(d); err != nil {
		return err
	}
	fmt.Printf("  snoozed  interviews paused for %s\n", d)
	return nil
}

func runResume(cmd *cobra.Command, args []string) error {
	if err := config.Resume(); err != nil {
		return err
	}
	fmt.Println("  resumed  interviews re-enabled")
	return nil
}

func runDisable(cmd *cobra.Command, args []string) error {
	if err := config.EnsureGritDir(); err != nil {
		return err
	}
	if err := config.Disable(); err != nil {
		return err
	}
	fmt.Println("  disabled interviews disabled until 'grit resume'")
	return nil
}
