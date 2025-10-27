package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/braginantonev/mhserver/internal/application"
	"github.com/braginantonev/mhserver/internal/server"
	"github.com/braginantonev/mhserver/internal/server/handlers/auth"
	auth_hand "github.com/braginantonev/mhserver/internal/server/handlers/auth"
	auth_mid "github.com/braginantonev/mhserver/internal/server/middlewares/auth"
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

	auth_handler, err := auth_hand.NewAuthHandler(auth.Config{
		DB:           app.DB,
		JWTSignature: app.JWTSignature,
	})
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	auth_middlewares := auth_mid.NewAuthMiddleware(auth_mid.Config{
		JWTSignature: app.JWTSignature,
	})

	srv := server.NewServer(
		auth_handler,
		auth_middlewares,
		nil,
	)

	if err = srv.Run(fmt.Sprintf("%s:%s", app.SubServers["main"].IP, app.SubServers["main"].Port)); err != nil {
		os.Exit(1)
	}

}
