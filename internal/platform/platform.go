package platform

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const wslErrorMessage = "This CLI is meant to be run only inside a WSL instance with access to powershell.exe"

// CheckWSLEnvironment verifies we're running inside WSL and that powershell.exe is accessible.
// Declared as a var so tests can override it without needing real WSL binaries.
var CheckWSLEnvironment = func() error {
	// Try wslinfo with known flags first
	if err := exec.Command("wslinfo", "--wsl-version").Run(); err == nil {
		return nil
	}
	if err := exec.Command("wslinfo", "--version").Run(); err == nil {
		return nil
	}
	// Fallback: look for "WSL2" in /proc/version
	if data, err := os.ReadFile("/proc/version"); err == nil {
		if strings.Contains(strings.ToUpper(string(data)), "WSL2") {
			return nil
		}
	}
	return fmt.Errorf("%s", wslErrorMessage)
}

// CheckWSLInterop verifies that WSL interop is enabled by checking the WSL_INTEROP environment variable.
// Declared as a var so tests can override it.
var CheckWSLInterop = func() error {
	if os.Getenv("WSL_INTEROP") == "" {
		return fmt.Errorf("WSL interoperability is disabled. Enable it in /etc/wsl.conf, see https://learn.microsoft.com/en-us/windows/wsl/wsl-config#example-wslconf-file for details.")
	}
	return nil
}
