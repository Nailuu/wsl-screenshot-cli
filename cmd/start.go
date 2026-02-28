package cmd

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	"nailu/wsl-screenshot-cli/internal/daemon"
	"nailu/wsl-screenshot-cli/internal/platform"
	"nailu/wsl-screenshot-cli/internal/poller"
)

var interval int
var outputDir string
var foreground bool

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the clipboard polling process",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if interval < 250 || interval > 5000 {
			return fmt.Errorf("Interval must be between 250 and 5000 ms (got %d)", interval)
		}
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("Output directory is not writable: %w", err)
		}
		if !foreground {
			if err := platform.CheckWSLEnvironment(); err != nil {
				return err
			}
			return daemon.Start(interval, outputDir)
		}
		return daemon.RunForeground(cmd.Context(), interval, outputDir, func(ctx context.Context, logger *log.Logger) error {
			return poller.Run(ctx, logger, interval, outputDir)
		})
	},
}

func init() {
	rootCmd.AddCommand(startCmd)

	startCmd.Flags().IntVarP(&interval, "interval", "i", 500, "Clipboard polling interval in ms (250-5000)")
	startCmd.Flags().StringVarP(&outputDir, "output", "o", "/tmp/.wsl-screenshot-cli/", "Directory to store PNGs")
	startCmd.Flags().BoolVar(&foreground, "foreground", false, "Run in foreground (used internally)")
	startCmd.Flags().MarkHidden("foreground")
}
