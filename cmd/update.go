package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/nailuu/wsl-screenshot-cli/internal/daemon"
)

const installScriptURL = "https://raw.githubusercontent.com/Nailuu/wsl-screenshot-cli/main/scripts/install.sh"

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update wsl-screenshot-cli to the latest version",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := exec.LookPath("curl"); err != nil {
			return fmt.Errorf("curl is required for updating but was not found in PATH")
		}

		daemonWasRunning := daemon.RunningPID() != 0
		if daemonWasRunning {
			fmt.Fprintln(cmd.OutOrStdout(), "Stopping running daemon before update...")
			daemon.Stop()
		}

		sh := exec.Command("bash", "-c", fmt.Sprintf("curl -fsSL %s | bash", installScriptURL))
		sh.Stdout = os.Stdout
		sh.Stderr = os.Stderr
		if err := sh.Run(); err != nil {
			return fmt.Errorf("update failed: %w", err)
		}

		if daemonWasRunning {
			fmt.Fprintln(cmd.OutOrStdout(), "\nDaemon was stopped for the update. Restart it with: wsl-screenshot-cli start --daemon")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
