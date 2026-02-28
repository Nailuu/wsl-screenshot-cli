package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "wsl-screenshot-cli",
	Short: "Monitor the Windows clipboard for screenshots and make them pasteable in WSL",
	Long: `wsl-screenshot-cli monitors the Windows clipboard for screenshots and converts
them into PNG files that can be pasted in both WSL and Windows applications.

How it works:
  The tool polls the clipboard at a configurable interval. When a new bitmap
  is detected (via hash deduplication), it converts the image to PNG and
  stores it locally.

It then sets three clipboard formats:
  CF_UNICODETEXT  — WSL path to the PNG, so you can paste in WSL terminals
  CF_BITMAP       — the PNG image data, so image paste still works
  CF_HDROP        — Windows UNC path (\\wsl.localhost\...) as a file drop,
                    so you can paste the file in Windows apps like Explorer

After a screenshot, you can paste the file path in a WSL terminal or paste
the file directly into Windows applications.`,
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
