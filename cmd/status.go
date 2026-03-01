package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/nailuu/wsl-screenshot-cli/internal/daemon"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of the clipboard polling process",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		w := cmd.OutOrStdout()
		info := daemon.Status()
		if info == nil {
			fmt.Fprintln(w, "Status:  not running")
			return
		}

		fmt.Fprintf(w, "Status:       running\n")
		fmt.Fprintf(w, "PID:          %d\n", info.PID)
		fmt.Fprintf(w, "Uptime:       %s\n", formatDuration(info.Uptime))
		fmt.Fprintf(w, "CPU usage:    %.1f%%\n", info.CPUPercent())
		fmt.Fprintf(w, "Memory:       %.1f MB\n", float64(info.MemoryRSSKB)/1024.0)
		fmt.Fprintf(w, "Screenshots:  %d\n", info.Screenshots)
		fmt.Fprintf(w, "Output dir:   %s\n", info.OutputDir)
		fmt.Fprintf(w, "Log file:     %s\n", info.LogFile)
	},
}

// formatDuration formats a duration as "Xh Ym Zs", omitting zero leading components.
func formatDuration(d time.Duration) string {
	totalSeconds := int(d.Seconds())
	if totalSeconds < 1 {
		return "0s"
	}
	h := totalSeconds / 3600
	m := (totalSeconds % 3600) / 60
	s := totalSeconds % 60

	switch {
	case h > 0:
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	case m > 0:
		return fmt.Sprintf("%dm %ds", m, s)
	default:
		return fmt.Sprintf("%ds", s)
	}
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
