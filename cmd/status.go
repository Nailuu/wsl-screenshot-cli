package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"nailu/wsl-screenshot-cli/internal/daemon"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of the clipboard polling process",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if pid := daemon.RunningPID(); pid != 0 {
			fmt.Printf("Polling process running (PID %d)\n", pid)
		} else {
			fmt.Println("Polling process is not running")
		}
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
