package main

import (
	"log/slog"
	"os"

	"github.com/braginantonev/mhserver/internal/application"
)

var (
	ArgAppMode = map[string]application.ApplicationMode{
		"-M": application.AppMode_MainServerOnly,
		"-S": application.AppMode_SubServersOnly,
	}
)

func main() {
	app := application.NewApplication()

	app_mode := application.AppMode_AllServers
	for _, arg := range os.Args {
		mode, ok := ArgAppMode[arg]
		if ok {
			app_mode = mode
		}
	}

	if err := app.Run(app_mode); err != nil {
		slog.Error("Failed run application", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
