package main

import (
	"context"
	"log/slog"
	"os"
)

func main() {
	ctx := context.Background()

	app := NewApp()

	err := app.Run(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to run app", "error", err)
		os.Exit(1)
	}
}
