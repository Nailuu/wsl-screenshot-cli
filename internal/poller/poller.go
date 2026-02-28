package poller

import (
	"context"
	"log"
	"time"
)

// Run polls the clipboard at the given interval until the context is cancelled.
func Run(ctx context.Context, logger *log.Logger, interval int, outputDir string) error {
	ticker := time.NewTicker(time.Duration(interval) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Println("Polling process shutting down...")
			return nil
		case <-ticker.C:
			// TODO: clipboard polling logic
		}
	}
}
