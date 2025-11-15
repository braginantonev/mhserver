package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/braginantonev/mhserver/internal/application"
	"github.com/braginantonev/mhserver/internal/server"
	"github.com/braginantonev/mhserver/internal/server/services/auth"
	auth_handlers "github.com/braginantonev/mhserver/internal/server/services/auth/handlers"
	auth_middlewares "github.com/braginantonev/mhserver/internal/server/services/auth/middlewares"
)

func main() {
	app := application.NewApplication()
	if err := app.InitDB(); err != nil {
		slog.Error(err.Error())
	}

	//* Setup auth service
	auth_handler := auth_handlers.NewAuthHandler(auth_handlers.Config{
		DB:           app.DB,
		JWTSignature: app.JWTSignature,
	})

	auth_middleware := auth_middlewares.NewAuthMiddleware(auth_middlewares.Config{
		JWTSignature: app.JWTSignature,
	})

	auth_service := auth.NewAuthService(auth_handler, auth_middleware)

	srv := server.NewServer(
		auth_service,
		nil,
	)

	if err := srv.Run(fmt.Sprintf("%s:%s", app.SubServers["main"].IP, app.SubServers["main"].Port)); err != nil {
		os.Exit(1)
	}

}
