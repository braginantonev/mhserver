package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/braginantonev/mhserver/internal/application"
	"github.com/braginantonev/mhserver/internal/server"
	"github.com/braginantonev/mhserver/internal/server/services/auth"
	amid "github.com/braginantonev/mhserver/internal/server/services/auth/middlewares"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file", err)
	}

	app := application.NewApplication()
	if err = app.InitDB(); err != nil {
		slog.Error(err.Error())
	}

	auth_service, err := auth.NewAuthService(auth.Config{
		DB:           app.DB,
		JWTSignature: app.JWTSignature,
	}, amid.NewAuthMiddleware(amid.Config{JWTSignature: app.JWTSignature}))

	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	srv := server.NewServer(
		auth_service,
		nil,
	)

	if err = srv.Run(fmt.Sprintf("%s:%s", app.SubServers["main"].IP, app.SubServers["main"].Port)); err != nil {
		os.Exit(1)
	}

}
