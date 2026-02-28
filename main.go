package main

import (
	"context"
	"os/signal"
	"syscall"

	"nailu/wsl-screenshot-cli/cmd"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	cmd.ExecuteContext(ctx)
}
