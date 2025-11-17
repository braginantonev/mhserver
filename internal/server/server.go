package server

import (
	"log/slog"
	"net/http"

	"github.com/braginantonev/mhserver/internal/server/services/auth"
	"github.com/braginantonev/mhserver/internal/server/services/data"
)

const (
	LOGIN_ENDPOINT    string = "/api/users/login"
	REGISTER_ENDPOINT string = "/api/users/register"
)

type Services struct {
	AuthService *auth.AuthService
	DataService *data.DataService
}

type Server struct {
	Services
}

func NewServer(
	auth_service *auth.AuthService,
	data_service *data.DataService,
) Server {
	return Server{
		Services: Services{
			AuthService: auth_service,
			DataService: data_service,
		},
	}
}

func (s Server) Run(addr string) error {
	mux := http.NewServeMux()

	mux.HandleFunc(LOGIN_ENDPOINT, s.AuthService.Handlers.Login)
	mux.HandleFunc(REGISTER_ENDPOINT, s.AuthService.Handlers.Register)

	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error(err.Error())
		return ErrFailedStartServer
	}

	return nil
}
