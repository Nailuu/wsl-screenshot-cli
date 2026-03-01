package cmd

import (
	"fmt"
	"testing"

	"github.com/nailuu/wsl-screenshot-cli/internal/platform"
)

func TestStart_DaemonFailsOnWSLCheckError(t *testing.T) {
	origCheck := platform.CheckWSLEnvironment
	defer func() { platform.CheckWSLEnvironment = origCheck }()

	wslErr := fmt.Errorf("not a WSL environment")
	platform.CheckWSLEnvironment = func() error { return wslErr }

	// Reset flags to defaults before test
	interval = 250
	outputDir = t.TempDir()
	daemonize = true
	verbose = false

	err := startCmd.RunE(startCmd, nil)
	if err == nil {
		t.Fatal("expected error from WSL check, got nil")
	}
	if err.Error() != wslErr.Error() {
		t.Errorf("expected WSL error %q, got %q", wslErr, err)
	}
}

func TestStart_InvalidInterval(t *testing.T) {
	tests := []struct {
		name     string
		interval int
	}{
		{"too_low", 50},
		{"too_high", 6000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interval = tt.interval
			outputDir = t.TempDir()
			daemonize = false
			verbose = false

			err := startCmd.RunE(startCmd, nil)
			if err == nil {
				t.Fatalf("expected error for interval %d, got nil", tt.interval)
			}
		})
	}
}
