package server

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/braginantonev/mhserver/internal/http/auth"
	"github.com/braginantonev/mhserver/internal/http/data"
)

const (
	// Auth
	LOGIN_ENDPOINT    string = "/api/users/login"
	REGISTER_ENDPOINT string = "/api/users/register"

	// Data
	SAVE_DATA_ENDPOINT    string = "/api/files/save"
	GET_DATA_ENDPOINT     string = "/api/files/get"
	GET_DATA_SUM_ENDPOINT string = "/api/files/sum"
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

func (s Server) Serve(ip, port string) error {
	mux := http.NewServeMux()

	// Auth
	mux.HandleFunc(LOGIN_ENDPOINT, s.AuthService.Handlers.Login)
	mux.HandleFunc(REGISTER_ENDPOINT, s.AuthService.Handlers.Register)

	// Data
	mux.HandleFunc(GET_DATA_ENDPOINT, s.AuthService.Middlewares.WithAuth(s.DataService.Handler.GetData))
	mux.HandleFunc(SAVE_DATA_ENDPOINT, s.AuthService.Middlewares.WithAuth(s.DataService.Handler.SaveData))
	mux.HandleFunc(GET_DATA_SUM_ENDPOINT, s.AuthService.Middlewares.WithAuth(s.DataService.Handler.GetSum))

	if err := http.ListenAndServe(fmt.Sprintf("%s:%s", ip, port), mux); err != nil {
		slog.Error(err.Error())
		return ErrFailedStartServer
	}

	return nil
}
