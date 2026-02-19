package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
)

func main() {
	ctx := context.Background()

	err := run(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to run", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	app, err := NewApp(ctx)
	if err != nil {
		return fmt.Errorf("failed to create app: %w", err)
	}

	err = app.Run(ctx)
	if err != nil {
		return fmt.Errorf("failed to run app: %w", err)
	}

	return nil
}
