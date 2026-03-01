package cmd

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/nailuu/wsl-screenshot-cli/internal/clipboard"
	"github.com/nailuu/wsl-screenshot-cli/internal/daemon"
	"github.com/nailuu/wsl-screenshot-cli/internal/platform"
	"github.com/nailuu/wsl-screenshot-cli/internal/poller"
)

var interval int
var outputDir string
var daemonize bool
var verbose bool

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the clipboard polling process",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if interval < 100 || interval > 5000 {
			return fmt.Errorf("Interval must be between 100 and 5000 ms (got %d)", interval)
		}
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("Output directory is not writable: %w", err)
		}
		if daemonize {
			if err := platform.CheckWSLEnvironment(); err != nil {
				return err
			}
			return daemon.Daemonize(interval, outputDir, verbose)
		}
		return daemon.Run(cmd.Context(), interval, outputDir, func(ctx context.Context, logger *log.Logger) error {
			return poller.Run(ctx, logger, interval, outputDir, func() (poller.Clipboard, error) {
				return clipboard.NewClient(logger, verbose)
			})
		})
	},
}

func init() {
	rootCmd.AddCommand(startCmd)

	startCmd.Flags().IntVarP(&interval, "interval", "i", 250, "Clipboard polling interval in ms (100-5000)")
	startCmd.Flags().StringVarP(&outputDir, "output", "o", "/tmp/.wsl-screenshot-cli/", "Directory to store PNGs")
	startCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Log all PowerShell I/O for debugging")
	startCmd.Flags().BoolVarP(&daemonize, "daemon", "d", false, "Run as a background daemon")
}
