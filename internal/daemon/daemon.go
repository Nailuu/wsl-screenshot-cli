package daemon

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

const PidFile = "/tmp/.wsl-screenshot-cli.pid"
const LogFile = "/tmp/.wsl-screenshot-cli.log"

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

// Start launches the daemon as a detached background process via re-exec.
func Start(interval int, outputDir string) error {
	if pid := RunningPID(); pid != 0 {
		fmt.Printf("Polling process is already running (PID %d)\n", pid)
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("Failed to get executable path: %w", err)
	}

	args := []string{"start", "--foreground",
		"--interval", strconv.Itoa(interval),
		"--output", outputDir,
	}

	child := exec.Command(exe, args...)
	child.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

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

	fmt.Printf("Polling process started with %d ms interval (PID %d). Logging to %s and saving screenshots to %s\n", interval, child.Process.Pid, LogFile, outputDir)
	return nil
}

// RunForeground writes the PID file, runs pollFn, and cleans up on exit.
func RunForeground(ctx context.Context, interval int, outputDir string, pollFn func(ctx context.Context, logger *log.Logger) error) error {
	if pid := RunningPID(); pid != 0 {
		fmt.Printf("Polling process is already running (PID %d)\n", pid)
		return nil
	}

	if err := os.WriteFile(PidFile, []byte(strconv.Itoa(os.Getpid())), 0644); err != nil {
		return fmt.Errorf("Failed to write PID file: %w", err)
	}
	defer os.Remove(PidFile)

	logger := log.New(os.Stdout, "", log.LstdFlags)
	logger.Printf("Polling process started with %d ms interval (PID %d)", interval, os.Getpid())
	return pollFn(ctx, logger)
}

// Stop sends SIGTERM to the running daemon and cleans up the PID file.
func Stop() {
	data, err := os.ReadFile(PidFile)
	if err != nil {
		fmt.Println("Polling process is not running")
		return
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		os.Remove(PidFile)
		fmt.Println("Polling process is not running. Cleaned up corrupt PID file.")
		return
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		os.Remove(PidFile)
		fmt.Println("Polling process is not running. Cleaned up stale PID file.")
		return
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		os.Remove(PidFile)
		fmt.Printf("Polling process was not running (PID %d). Cleaned up stale PID file.\n", pid)
		return
	}

	os.Remove(PidFile)
	fmt.Printf("Polling process stopped successfully (PID %d)\n", pid)
}
