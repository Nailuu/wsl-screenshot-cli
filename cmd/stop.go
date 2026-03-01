package cmd

import (
	"github.com/spf13/cobra"

	"github.com/nailuu/wsl-screenshot-cli/internal/daemon"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the clipboard polling process",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		daemon.Stop()
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
