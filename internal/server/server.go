package server

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/braginantonev/mhserver/internal/domain"
)

const (
	// Auth
	LOGIN_ENDPOINT    string = "/api/users/login"
	REGISTER_ENDPOINT string = "/api/users/register"

	// Data
	CREATE_CONNECTION_ENDPOINT string = "/api/files/conn"
	SAVE_DATA_ENDPOINT         string = "/api/files/save"
	GET_DATA_ENDPOINT          string = "/api/files/get"
	GET_DATA_SUM_ENDPOINT      string = "/api/files/sum"
)

type Server struct {
	AuthService *domain.HttpAuthService
	DataService *domain.HttpDataService
}

func (s Server) Serve(ip, port string) error {
	mux := http.NewServeMux()

	// Auth
	mux.HandleFunc(LOGIN_ENDPOINT, s.AuthService.Handlers.Login)
	mux.HandleFunc(REGISTER_ENDPOINT, s.AuthService.Handlers.Register)

	// Data
	mux.HandleFunc(CREATE_CONNECTION_ENDPOINT, s.AuthService.Middlewares.WithAuth(s.DataService.Handler.CreateConnection))
	mux.HandleFunc(GET_DATA_ENDPOINT, s.AuthService.Middlewares.WithAuth(s.DataService.Handler.GetData))
	mux.HandleFunc(SAVE_DATA_ENDPOINT, s.AuthService.Middlewares.WithAuth(s.DataService.Handler.SaveData))
	mux.HandleFunc(GET_DATA_SUM_ENDPOINT, s.AuthService.Middlewares.WithAuth(s.DataService.Handler.GetSum))

	if err := http.ListenAndServe(fmt.Sprintf("%s:%s", ip, port), mux); err != nil {
		slog.Error(err.Error())
		return ErrFailedStartServer
	}

	return nil
}
