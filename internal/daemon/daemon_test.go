package daemon

import (
	"bytes"
	"context"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// setTestPaths overrides package-level vars to use a temp dir for isolation.
// Returns a cleanup function that restores the original values.
func setTestPaths(t *testing.T) func() {
	t.Helper()
	tmp := t.TempDir()
	origPid := PidFile
	origLog := LogFile
	origState := StateFile
	origDefault := DefaultOutputDir
	origOutput := Output

	PidFile = filepath.Join(tmp, "test.pid")
	LogFile = filepath.Join(tmp, "test.log")
	StateFile = filepath.Join(tmp, "test.state")
	DefaultOutputDir = filepath.Join(tmp, "output") + "/"
	Output = io.Discard

	return func() {
		PidFile = origPid
		LogFile = origLog
		StateFile = origState
		DefaultOutputDir = origDefault
		Output = origOutput
	}
}

func TestCountScreenshots(t *testing.T) {
	t.Run("empty_dir", func(t *testing.T) {
		dir := t.TempDir()
		if got := countScreenshots(dir); got != 0 {
			t.Errorf("countScreenshots(empty) = %d, want 0", got)
		}
	})

	t.Run("nonexistent_dir", func(t *testing.T) {
		if got := countScreenshots("/nonexistent/path"); got != 0 {
			t.Errorf("countScreenshots(nonexistent) = %d, want 0", got)
		}
	})

	t.Run("mixed_files", func(t *testing.T) {
		dir := t.TempDir()
		for _, name := range []string{"a.png", "b.png", "c.txt", "d.jpg"} {
			os.WriteFile(filepath.Join(dir, name), []byte("x"), 0644)
		}
		if got := countScreenshots(dir); got != 2 {
			t.Errorf("countScreenshots(mixed) = %d, want 2", got)
		}
	})
}

func TestReadOutputDir(t *testing.T) {
	cleanup := setTestPaths(t)
	defer cleanup()

	t.Run("missing_state_file", func(t *testing.T) {
		os.Remove(StateFile)
		if got := readOutputDir(); got != DefaultOutputDir {
			t.Errorf("readOutputDir() = %q, want %q", got, DefaultOutputDir)
		}
	})

	t.Run("empty_state_file", func(t *testing.T) {
		os.WriteFile(StateFile, []byte("  \n"), 0644)
		if got := readOutputDir(); got != DefaultOutputDir {
			t.Errorf("readOutputDir() = %q, want %q", got, DefaultOutputDir)
		}
	})

	t.Run("valid_state_file", func(t *testing.T) {
		os.WriteFile(StateFile, []byte("/custom/path"), 0644)
		if got := readOutputDir(); got != "/custom/path" {
			t.Errorf("readOutputDir() = %q, want %q", got, "/custom/path")
		}
	})
}

func TestRunningPID_NoPidFile(t *testing.T) {
	cleanup := setTestPaths(t)
	defer cleanup()

	if got := RunningPID(); got != 0 {
		t.Errorf("RunningPID() = %d, want 0 (no PID file)", got)
	}
}

func TestRunningPID_CurrentProcess(t *testing.T) {
	cleanup := setTestPaths(t)
	defer cleanup()

	os.WriteFile(PidFile, []byte(strconv.Itoa(os.Getpid())), 0644)
	if got := RunningPID(); got != os.Getpid() {
		t.Errorf("RunningPID() = %d, want %d", got, os.Getpid())
	}
}

func TestRunningPID_StalePid(t *testing.T) {
	cleanup := setTestPaths(t)
	defer cleanup()

	os.WriteFile(PidFile, []byte("999999"), 0644)
	if got := RunningPID(); got != 0 {
		t.Errorf("RunningPID() = %d, want 0 (stale PID)", got)
	}
	// PID file should be cleaned up
	if _, err := os.Stat(PidFile); !os.IsNotExist(err) {
		t.Error("stale PID file was not cleaned up")
	}
}

func TestRunningPID_CorruptFile(t *testing.T) {
	cleanup := setTestPaths(t)
	defer cleanup()

	os.WriteFile(PidFile, []byte("not-a-number"), 0644)
	if got := RunningPID(); got != 0 {
		t.Errorf("RunningPID() = %d, want 0 (corrupt PID file)", got)
	}
	if _, err := os.Stat(PidFile); !os.IsNotExist(err) {
		t.Error("corrupt PID file was not cleaned up")
	}
}

func TestRun_PidAndStateLifecycle(t *testing.T) {
	cleanup := setTestPaths(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	outputDir := t.TempDir()

	pollStarted := make(chan struct{})
	done := make(chan error, 1)

	go func() {
		done <- Run(ctx, 250, outputDir, func(ctx context.Context, logger *log.Logger) error {
			close(pollStarted)
			<-ctx.Done()
			return nil
		})
	}()

	// Wait for pollFn to start
	select {
	case <-pollStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("pollFn never started")
	}

	// PID and state files should exist during run
	if _, err := os.Stat(PidFile); err != nil {
		t.Errorf("PID file should exist during run: %v", err)
	}
	if _, err := os.Stat(StateFile); err != nil {
		t.Errorf("state file should exist during run: %v", err)
	}

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not exit after context cancel")
	}

	// Both files should be cleaned up after Run returns
	if _, err := os.Stat(PidFile); !os.IsNotExist(err) {
		t.Error("PID file should be removed after Run exits")
	}
	if _, err := os.Stat(StateFile); !os.IsNotExist(err) {
		t.Error("state file should be removed after Run exits")
	}
}

func TestRun_AlreadyRunning(t *testing.T) {
	cleanup := setTestPaths(t)
	defer cleanup()

	// Write our own PID as the running process
	os.WriteFile(PidFile, []byte(strconv.Itoa(os.Getpid())), 0644)

	pollCalled := false
	err := Run(context.Background(), 250, t.TempDir(), func(ctx context.Context, logger *log.Logger) error {
		pollCalled = true
		return nil
	})

	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if pollCalled {
		t.Error("pollFn should not be called when daemon is already running")
	}
}

func TestStop_SendsSIGTERM(t *testing.T) {
	cleanup := setTestPaths(t)
	defer cleanup()

	// Start a sleep subprocess
	child := exec.Command("sleep", "60")
	if err := child.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	pid := child.Process.Pid

	os.WriteFile(PidFile, []byte(strconv.Itoa(pid)), 0644)

	Stop()

	// Wait for the process to actually exit
	waitDone := make(chan error, 1)
	go func() { waitDone <- child.Wait() }()

	select {
	case <-waitDone:
		// Process terminated
	case <-time.After(5 * time.Second):
		child.Process.Kill()
		t.Fatal("sleep process did not terminate after Stop()")
	}

	// PID file should be cleaned up
	if _, err := os.Stat(PidFile); !os.IsNotExist(err) {
		t.Error("PID file should be removed after Stop()")
	}
}

func TestStop_NotRunning(t *testing.T) {
	cleanup := setTestPaths(t)
	defer cleanup()

	// No PID file — should not panic
	Stop()

	// With stale PID
	os.WriteFile(PidFile, []byte("999999"), 0644)
	Stop()

	// With corrupt PID
	os.WriteFile(PidFile, []byte("garbage"), 0644)
	Stop()
}

// TestHelperProcess is invoked as a fake daemon subprocess.
// It exits after a short sleep to simulate a daemon that started successfully.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	time.Sleep(100 * time.Millisecond)
	os.Exit(0)
}

// helperDaemonCmd returns a newDaemonCmd override that spawns a TestHelperProcess
// instead of re-execing the real binary.
func helperDaemonCmd(t *testing.T) func(int, string, bool) (*exec.Cmd, error) {
	t.Helper()
	return func(interval int, outputDir string, verbose bool) (*exec.Cmd, error) {
		cmd := exec.Command(os.Args[0], "-test.run=^TestHelperProcess$")
		cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		return cmd, nil
	}
}

func TestDaemonize_StartsProcess(t *testing.T) {
	cleanup := setTestPaths(t)
	defer cleanup()

	orig := newDaemonCmd
	defer func() { newDaemonCmd = orig }()
	newDaemonCmd = helperDaemonCmd(t)

	var buf bytes.Buffer
	Output = &buf

	err := Daemonize(250, t.TempDir(), false)
	if err != nil {
		t.Fatalf("Daemonize() error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Polling process started") {
		t.Errorf("expected success message, got: %q", out)
	}

	// Log file should have been created
	if _, err := os.Stat(LogFile); err != nil {
		t.Errorf("log file should exist after Daemonize: %v", err)
	}
}

func TestDaemonize_AlreadyRunning(t *testing.T) {
	cleanup := setTestPaths(t)
	defer cleanup()

	// Write our own PID as the running process
	os.WriteFile(PidFile, []byte(strconv.Itoa(os.Getpid())), 0644)

	var buf bytes.Buffer
	Output = &buf

	err := Daemonize(250, t.TempDir(), false)
	if err != nil {
		t.Fatalf("Daemonize() error: %v", err)
	}

	if !strings.Contains(buf.String(), "already running") {
		t.Errorf("expected 'already running' message, got: %q", buf.String())
	}
}
