package cmd

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/nailuu/wsl-screenshot-cli/internal/clipboard"
	"github.com/nailuu/wsl-screenshot-cli/internal/daemon"
	"github.com/nailuu/wsl-screenshot-cli/internal/platform"
	"github.com/nailuu/wsl-screenshot-cli/internal/poller"
	versioncheck "github.com/nailuu/wsl-screenshot-cli/internal/version"
)

var interval int
var outputDir string
var daemonize bool
var verbose bool
var quiet bool

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the clipboard polling process",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if quiet {
			daemon.Output = io.Discard
		}

		if latest, err := versioncheck.CheckForUpdate(version); err == nil && latest != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "\nNew update available (v%s), run `wsl-screenshot-cli update` to install it.\n\n", latest)
		}

		if interval < 100 || interval > 5000 {
			return fmt.Errorf("Interval must be between 100 and 5000 ms (got %d)", interval)
		}

		if err := os.MkdirAll(outputDir, 0750); err != nil {
			return fmt.Errorf("Output directory is not writable: %w", err)
		}

		if err := platform.CheckWSLEnvironment(); err != nil {
			return err
		}

		if err := platform.CheckWSLInterop(); err != nil {
			return err
		}

		if daemonize {
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
	startCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Suppress informational messages")
}
