package platform

import (
	"fmt"
	"os"
	"os/exec"
)

const wslErrorMessage = "This CLI is meant to be run only inside a WSL instance with access to powershell.exe"

// CheckWSLEnvironment verifies we're running inside WSL and that powershell.exe is accessible.
// Declared as a var so tests can override it without needing real WSL binaries.
var CheckWSLEnvironment = func() error {
	if err := exec.Command("wslinfo", "--wsl-version").Run(); err != nil {
		if err := exec.Command("wslinfo", "--version").Run(); err != nil {
			return fmt.Errorf("%s", wslErrorMessage)
		}
	}
	return nil
}

// CheckWSLInterop verifies that WSL interop is enabled by checking the WSL_INTEROP environment variable.
// Declared as a var so tests can override it.
var CheckWSLInterop = func() error {
	if os.Getenv("WSL_INTEROP") == "" {
		return fmt.Errorf("WSL interoperability is disabled. Enable it in /etc/wsl.conf, see https://learn.microsoft.com/en-us/windows/wsl/wsl-config#example-wslconf-file for details.")
	}
	return nil
}
