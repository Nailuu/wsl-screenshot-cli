package cmd

import (
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"zero", 0, "0s"},
		{"sub_second", 500 * time.Millisecond, "0s"},
		{"one_second", time.Second, "1s"},
		{"seconds_only", 45 * time.Second, "45s"},
		{"minutes_and_seconds", 3*time.Minute + 12*time.Second, "3m 12s"},
		{"exact_minutes", 5 * time.Minute, "5m 0s"},
		{"hours_minutes_seconds", 2*time.Hour + 15*time.Minute + 30*time.Second, "2h 15m 30s"},
		{"negative", -5 * time.Second, "0s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(tt.duration)
			if got != tt.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, got, tt.want)
			}
		})
	}
}
