package poller

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const maxConsecutiveErrors = 5

// Clipboard abstracts clipboard operations for testability.
type Clipboard interface {
	Check() ([]byte, error)
	UpdateClipboard(wslPath, winPath string) error
	Close() error
}

// ClientFactory creates a new Clipboard client.
type ClientFactory func() (Clipboard, error)

// Run polls the clipboard at the given interval until the context is cancelled.
func Run(ctx context.Context, logger *log.Logger, interval int, outputDir string, newClient ClientFactory) error {
	client, err := newClient()
	if err != nil {
		return fmt.Errorf("start clipboard client: %w", err)
	}
	defer func() { _ = client.Close() }()

	ticker := time.NewTicker(time.Duration(interval) * time.Millisecond)
	defer ticker.Stop()

	consecutiveErrors := 0

	for {
		select {
		case <-ctx.Done():
			logger.Println("Polling process shutting down...")
			return nil
		case <-ticker.C:
			if err := poll(client, logger, outputDir); err != nil {
				consecutiveErrors++
				logger.Printf("Poll error (%d/%d): %v", consecutiveErrors, maxConsecutiveErrors, err)

				if consecutiveErrors >= maxConsecutiveErrors {
					logger.Println("Too many consecutive errors, restarting PowerShell client...")
					_ = client.Close()

					client, err = newClient()
					if err != nil {
						return fmt.Errorf("restart clipboard client: %w", err)
					}
					consecutiveErrors = 0
				}
			} else {
				consecutiveErrors = 0
			}
		}
	}
}

// poll performs a single clipboard check cycle: check -> hash -> dedup -> save -> update.
func poll(client Clipboard, logger *log.Logger, outputDir string) error {
	pngData, err := client.Check()
	if err != nil {
		return fmt.Errorf("check clipboard: %w", err)
	}
	if pngData == nil {
		return nil // no image in clipboard
	}

	hash := hashBytes(pngData)
	filename := hash + ".png"
	filePath := filepath.Join(outputDir, filename)

	// Only write if file doesn't already exist (content-addressable dedup).
	// We intentionally do NOT return early when the file exists because actions
	// like Snipping Tool's Copy button or Undo button overwrite the clipboard
	// with just CF_BITMAP, stripping our 3-format fingerprint (CF_BITMAP +
	// CF_UNICODETEXT + CF_HDROP). The SHA256 match tells us the image is already
	// saved locally, so we skip the write but still fall through to
	// UpdateClipboard below to restore the useful text-path and file-drop formats.
	if _, err := os.Stat(filePath); err != nil {
		if err := os.WriteFile(filePath, pngData, 0644); err != nil { // #nosec G306 -- screenshots must be readable by Windows apps via WSL interop
			return fmt.Errorf("write %s: %w", filename, err)
		}
		logger.Printf("New screenshot saved: %s (%d bytes)", filename, len(pngData))
	}

	winPath, err := wslToWinPath(filePath)
	if err != nil {
		logger.Printf("Warning: wslpath failed, clipboard not updated: %v", err)
		return nil // file saved, just can't update clipboard
	}

	if err := client.UpdateClipboard(filePath, winPath); err != nil {
		logger.Printf("Warning: clipboard update failed: %v", err)
		return nil // file saved, just can't update clipboard
	}

	logger.Printf("Clipboard updated (WSL: %s)", filePath)
	return nil
}

// hashBytes returns the lowercase hex SHA256 of data.
func hashBytes(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}

// wslToWinPath converts a WSL path to a Windows path using wslpath -w.
// Declared as a var so tests can override it without needing the wslpath binary.
var wslToWinPath = func(wslPath string) (string, error) {
	out, err := exec.Command("wslpath", "-w", wslPath).Output() // #nosec G204 -- wslPath is cleaned by filepath.Join, argv-separated (no shell)
	if err != nil {
		return "", fmt.Errorf("wslpath -w %q: %w", wslPath, err)
	}
	return strings.TrimSpace(string(out)), nil
}
