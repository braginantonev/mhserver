package main

import (
	"log/slog"
	"os"

	"github.com/braginantonev/mhserver/internal/application"
)

func main() {
	app := application.NewApplication()
	if err := app.Run(application.AppMode_AllServers); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
