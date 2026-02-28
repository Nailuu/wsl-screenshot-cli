package platform

import (
	"fmt"
	"os/exec"
)

const wslErrorMessage = "This CLI is meant to be run only inside a WSL instance with access to powershell.exe"

// CheckWSLEnvironment verifies we're running inside WSL and that powershell.exe is accessible.
func CheckWSLEnvironment() error {
	// Check 1: verify we're inside WSL
	if err := exec.Command("wslinfo", "--version").Run(); err != nil {
		return fmt.Errorf("%s", wslErrorMessage)
	}

	// Check 2: verify powershell.exe is accessible and functional
	if err := exec.Command("powershell.exe", "-STA", "-NoLogo", "-NoProfile", "-NonInteractive", "-Command", "echo ok").Run(); err != nil {
		return fmt.Errorf("%s", wslErrorMessage)
	}

	return nil
}
