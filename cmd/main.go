package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/braginantonev/mhserver/internal/application"
	"github.com/braginantonev/mhserver/version"
)

var (
	ArgAppMode = map[string]application.ApplicationMode{
		"-M": application.AppMode_MainServerOnly,
		"-S": application.AppMode_SubServersOnly,
	}
)

func main() {
	fmt.Printf("Mhserver (ver. %s)\n", version.Version)

	app, err := application.NewApplication()
	if err != nil {
		slog.Error("failed init application", slog.Any("error", err))
		os.Exit(1)
	}

	app_mode := application.AppMode_AllServers
	for _, arg := range os.Args {
		mode, ok := ArgAppMode[arg]
		if ok {
			app_mode = mode
		}
	}

	if err := app.Run(app_mode); err != nil {
		slog.Error("Failed run application", slog.Any("error", err))
		os.Exit(1)
	}
}
