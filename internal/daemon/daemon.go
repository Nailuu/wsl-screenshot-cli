package daemon

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// Output is the writer for user-facing messages. Tests can set it to io.Discard.
var Output io.Writer = os.Stdout

var PidFile = "/tmp/.wsl-screenshot-cli.pid"
var LogFile = "/tmp/.wsl-screenshot-cli.log"
var StateFile = "/tmp/.wsl-screenshot-cli.state"
var DefaultOutputDir = "/tmp/.wsl-screenshot-cli/"

// readOutputDir reads the persisted output directory from the state file,
// falling back to DefaultOutputDir if the file is missing or empty.
func readOutputDir() string {
	data, err := os.ReadFile(StateFile)
	if err != nil {
		return DefaultOutputDir
	}
	dir := strings.TrimSpace(string(data))
	if dir == "" {
		return DefaultOutputDir
	}
	return dir
}

// RunningPID returns the PID of the running process, or 0 if not running.
// Cleans up stale PID files (e.g. after WSL restart).
func RunningPID() int {
	data, err := os.ReadFile(PidFile)
	if err != nil {
		return 0
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		os.Remove(PidFile)
		return 0
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		os.Remove(PidFile)
		return 0
	}

	// Signal 0 checks if the process is alive without actually sending a signal
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		os.Remove(PidFile) // stale PID file (e.g. after WSL restart), clean up
		return 0
	}

	return pid
}

// newDaemonCmd builds the exec.Cmd for the re-exec daemon process.
// Declared as a var so tests can override it with a fake process.
var newDaemonCmd = func(interval int, outputDir string, verbose bool) (*exec.Cmd, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("Failed to get executable path: %w", err)
	}

	args := []string{"start",
		"--interval", strconv.Itoa(interval),
		"--output", outputDir,
	}
	if verbose {
		args = append(args, "--verbose")
	}

	cmd := exec.Command(exe, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	return cmd, nil
}

// Daemonize launches a detached background process via re-exec.
func Daemonize(interval int, outputDir string, verbose bool) error {
	if pid := RunningPID(); pid != 0 {
		fmt.Fprintf(Output, "Polling process is already running (PID %d)\n", pid)
		return nil
	}

	child, err := newDaemonCmd(interval, outputDir, verbose)
	if err != nil {
		return err
	}

	logF, err := os.OpenFile(LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("Failed to open log file: %w", err)
	}
	child.Stdout = logF
	child.Stderr = logF

	if err := child.Start(); err != nil {
		logF.Close()
		return fmt.Errorf("Failed to start daemon: %w", err)
	}
	logF.Close()

	fmt.Fprintf(Output, "Polling process started (PID %d). Run 'wsl-screenshot-cli status' to check status.\n", child.Process.Pid)
	return nil
}

// Run writes the PID file, runs pollFn, and cleans up on exit.
func Run(ctx context.Context, interval int, outputDir string, pollFn func(ctx context.Context, logger *log.Logger) error) error {
	if pid := RunningPID(); pid != 0 {
		fmt.Fprintf(Output, "Polling process is already running (PID %d)\n", pid)
		return nil
	}

	if err := os.WriteFile(PidFile, []byte(strconv.Itoa(os.Getpid())), 0644); err != nil {
		return fmt.Errorf("Failed to write PID file: %w", err)
	}
	defer os.Remove(PidFile)

	if err := os.WriteFile(StateFile, []byte(outputDir), 0644); err != nil {
		return fmt.Errorf("Failed to write state file: %w", err)
	}
	defer os.Remove(StateFile)

	logger := log.New(Output, "", log.LstdFlags|log.Lmicroseconds)
	logger.Printf("Polling process started successfully (PID %d)", os.Getpid())
	return pollFn(ctx, logger)
}

// Stop sends SIGTERM to the running daemon and cleans up the PID file.
func Stop() {
	data, err := os.ReadFile(PidFile)
	if err != nil {
		fmt.Fprintln(Output, "Polling process is not running")
		return
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		os.Remove(PidFile)
		fmt.Fprintln(Output, "Polling process is not running. Cleaned up corrupt PID file.")
		return
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		os.Remove(PidFile)
		fmt.Fprintln(Output, "Polling process is not running. Cleaned up stale PID file.")
		return
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		os.Remove(PidFile)
		fmt.Fprintf(Output, "Polling process was not running (PID %d). Cleaned up stale PID file.\n", pid)
		return
	}

	os.Remove(PidFile)
	fmt.Fprintf(Output, "Polling process stopped successfully (PID %d)\n", pid)
}
