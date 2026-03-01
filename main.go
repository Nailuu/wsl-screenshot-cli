package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/nailuu/wsl-screenshot-cli/cmd"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	cmd.ExecuteContext(ctx)
}
