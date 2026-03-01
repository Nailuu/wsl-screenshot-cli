package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "wsl-screenshot-cli",
	Short: "Monitor the Windows clipboard for screenshots, making them pasteable in WSL while preserving Windows paste functionality",
	Long: `wsl-screenshot-cli monitors the Windows clipboard for screenshots, making
them pasteable in WSL (e.g. Claude Code CLI, Codex CLI, ...) while preserving
Windows paste functionality.

A persistent powershell.exe -STA subprocess handles all clipboard access
via a stdin/stdout text protocol. It polls at a configurable interval,
using GetClipboardSequenceNumber() to skip reads when nothing has changed.
When a new bitmap is detected, it saves the PNG (deduplicated by SHA256
hash) and sets three clipboard formats at once:

  CF_UNICODETEXT  — WSL path to the PNG, so you can paste in WSL terminals
  CF_BITMAP       — the original image data, preserving normal image paste
  CF_HDROP        — Windows UNC path as a file drop, preserving paste-as-file

After a screenshot, you can paste the file path in a WSL terminal and still
paste normally in Windows applications.`,
}

// ExecuteContext adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func ExecuteContext(ctx context.Context) {
	err := rootCmd.ExecuteContext(ctx)
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.SilenceUsage = true
}
