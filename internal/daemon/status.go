package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ProcessInfo holds diagnostic information about the running daemon.
type ProcessInfo struct {
	PID         int
	Uptime      time.Duration
	CPUTime     float64 // total user+system CPU seconds
	MemoryRSSKB int64   // resident set size in KB
	Screenshots int
	OutputDir   string
	LogFile     string
}

// CPUPercent returns the average CPU usage as a percentage over the process lifetime.
func (p *ProcessInfo) CPUPercent() float64 {
	uptimeSec := p.Uptime.Seconds()
	if uptimeSec <= 0 {
		return 0
	}
	return (p.CPUTime / uptimeSec) * 100
}

// Status returns process diagnostics if the daemon is running, or nil if not.
func Status() *ProcessInfo {
	pid := RunningPID()
	if pid == 0 {
		return nil
	}

	outputDir := readOutputDir()

	info := &ProcessInfo{
		PID:       pid,
		OutputDir: outputDir,
		LogFile:   LogFile,
	}

	info.Uptime = parseUptime(pid)
	info.CPUTime = parseCPUTime(pid)
	info.MemoryRSSKB = parseVmRSS(pid)
	info.Screenshots = countScreenshots(outputDir)

	return info
}

// parseUptime calculates how long the process has been running by comparing
// its start time (from /proc/<pid>/stat field 22) against system uptime.
func parseUptime(pid int) time.Duration {
	// Read system uptime
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return 0
	}
	systemUptime, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0
	}

	// Read process start time (field 22 in /proc/<pid>/stat, 1-indexed)
	data, err = os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0
	}
	// Fields after the comm field (which is in parens and may contain spaces)
	// Find the closing paren, then split the rest
	closeParen := strings.LastIndex(string(data), ")")
	if closeParen < 0 {
		return 0
	}
	rest := strings.Fields(string(data)[closeParen+2:]) // skip ") "
	// rest[0] = field 3 (state), so field 22 = rest[19]
	if len(rest) < 20 {
		return 0
	}
	startTicks, err := strconv.ParseInt(rest[19], 10, 64)
	if err != nil {
		return 0
	}

	clkTck := int64(100) // sysconf(_SC_CLK_TCK), 100 on virtually all Linux
	processStartSec := float64(startTicks) / float64(clkTck)
	uptimeSec := systemUptime - processStartSec

	if uptimeSec < 0 {
		return 0
	}
	return time.Duration(uptimeSec * float64(time.Second))
}

// parseCPUTime returns total user+system CPU time in seconds from /proc/<pid>/stat.
func parseCPUTime(pid int) float64 {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0
	}
	closeParen := strings.LastIndex(string(data), ")")
	if closeParen < 0 {
		return 0
	}
	rest := strings.Fields(string(data)[closeParen+2:])
	// rest[11] = field 14 (utime), rest[12] = field 15 (stime)
	if len(rest) < 13 {
		return 0
	}
	utime, err1 := strconv.ParseInt(rest[11], 10, 64)
	stime, err2 := strconv.ParseInt(rest[12], 10, 64)
	if err1 != nil || err2 != nil {
		return 0
	}
	clkTck := int64(100)
	return float64(utime+stime) / float64(clkTck)
}

// parseVmRSS reads the VmRSS line from /proc/<pid>/status and returns the value in KB.
func parseVmRSS(pid int) int64 {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "VmRSS:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				val, err := strconv.ParseInt(fields[1], 10, 64)
				if err == nil {
					return val
				}
			}
		}
	}
	return 0
}

// countScreenshots counts .png files in the given directory.
func countScreenshots(dir string) int {
	matches, err := filepath.Glob(filepath.Join(dir, "*.png"))
	if err != nil {
		return 0
	}
	return len(matches)
}
