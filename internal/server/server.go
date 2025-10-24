package server

import (
	"log/slog"
	"net/http"

	"github.com/braginantonev/mhserver/internal/server/handlers/auth"
	"github.com/braginantonev/mhserver/internal/server/handlers/data"
)

type Services struct {
	AuthService auth.AuthService
	DataService data.DataService
}

type Server struct {
	Services
}

func NewServer(
	auth_service auth.AuthService,
	data_service data.DataService,
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

	mux.HandleFunc("/api/users/login", s.AuthService.Login)
	mux.HandleFunc("/api/users/register", s.AuthService.Register)

	//Todo: mux.HandleFunc("/files/data", DataHandler(GetDataHandler, SaveDataHandler))
	//Todo: mux.HandleFunc("/files/data/hash", GetHashHandler)

	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error(err.Error())
		return ErrFailedStartServer
	}

	return nil
}
