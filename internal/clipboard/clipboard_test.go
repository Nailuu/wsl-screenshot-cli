package clipboard

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestHelperProcess is invoked by tests as a fake PowerShell subprocess.
// It speaks the same protocol: READY on start, CHECK→NONE/IMAGE, EXIT→exit.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	scanner := bufio.NewScanner(os.Stdin)

	// Send READY
	fmt.Println("READY")

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case line == "CHECK":
			behavior := os.Getenv("HELPER_CHECK_BEHAVIOR")
			switch behavior {
			case "IMAGE":
				imgData := []byte("fake-png-data-for-test")
				b64 := base64.StdEncoding.EncodeToString(imgData)
				fmt.Println("IMAGE")
				fmt.Println(b64)
				fmt.Println("END")
			default:
				fmt.Println("NONE")
			}
		case strings.HasPrefix(line, "UPDATE|"):
			fmt.Println("OK")
		case line == "EXIT":
			os.Exit(0)
		}
	}
	// stdin closed (pipe broken) — exit cleanly
	os.Exit(0)
}

// helperCommand returns a function that creates an exec.Cmd running
// the TestHelperProcess with the given environment.
func helperCommand(t *testing.T, envs ...string) func() *exec.Cmd {
	t.Helper()
	return func() *exec.Cmd {
		cmd := exec.Command(os.Args[0], "-test.run=^TestHelperProcess$")
		cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
		cmd.Env = append(cmd.Env, envs...)
		return cmd
	}
}

func TestNewClient_ReadyHandshake(t *testing.T) {
	orig := newPSCommand
	defer func() { newPSCommand = orig }()
	newPSCommand = helperCommand(t)

	logger := testLogger(t)
	client, err := NewClient(logger, false)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	defer client.Close()
}

func TestCheck_ReturnsNone(t *testing.T) {
	orig := newPSCommand
	defer func() { newPSCommand = orig }()
	newPSCommand = helperCommand(t)

	logger := testLogger(t)
	client, err := NewClient(logger, false)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	defer client.Close()

	data, err := client.Check()
	if err != nil {
		t.Fatalf("Check() error: %v", err)
	}
	if data != nil {
		t.Errorf("Check() = %v, want nil (NONE)", data)
	}
}

func TestCheck_ReturnsImage(t *testing.T) {
	orig := newPSCommand
	defer func() { newPSCommand = orig }()
	newPSCommand = helperCommand(t, "HELPER_CHECK_BEHAVIOR=IMAGE")

	logger := testLogger(t)
	client, err := NewClient(logger, false)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	defer client.Close()

	data, err := client.Check()
	if err != nil {
		t.Fatalf("Check() error: %v", err)
	}
	if data == nil {
		t.Fatal("Check() returned nil, expected image data")
	}
	if string(data) != "fake-png-data-for-test" {
		t.Errorf("Check() = %q, want %q", data, "fake-png-data-for-test")
	}
}

func TestClose_SendsEXIT(t *testing.T) {
	orig := newPSCommand
	defer func() { newPSCommand = orig }()
	newPSCommand = helperCommand(t)

	logger := testLogger(t)
	client, err := NewClient(logger, false)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	err = client.Close()
	if err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	// The process should have exited cleanly (exit code 0).
	// If Close() didn't send EXIT, the process would hang and Wait() would block.
}

func testLogger(t *testing.T) *log.Logger {
	t.Helper()
	return log.New(io.Discard, "", 0)
}
