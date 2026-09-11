package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/braginantonev/mhserver/internal/application"
	"github.com/braginantonev/mhserver/version"
)

func main() {
	fmt.Printf("Mhserver (ver. %s)\n", version.Version)

	app, err := application.NewApplication()
	if err != nil {
		slog.Error("failed init application", slog.Any("error", err))
		os.Exit(1)
	}

	ctx := context.Background()

	if err := app.Run(ctx); err != nil {
		slog.Error("Failed run application", slog.Any("error", err))
		os.Exit(1)
	}
}
