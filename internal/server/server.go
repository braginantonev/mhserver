package server

import (
	"log/slog"
	"net/http"

	auth_hand "github.com/braginantonev/mhserver/internal/server/handlers/auth"
	"github.com/braginantonev/mhserver/internal/server/handlers/data"
	auth_mid "github.com/braginantonev/mhserver/internal/server/middlewares/auth"
)

type Services struct {
	AuthHandleService auth_hand.AuthHandleService
	AuthMiddleService auth_mid.AuthMiddleService
	DataService       data.DataService
}

type Server struct {
	Services
}

func NewServer(
	auth_handlers_service auth_hand.AuthHandleService,
	auth_middlewares_service auth_mid.AuthMiddleService,
	data_service data.DataService,
) Server {
	return Server{
		Services: Services{
			AuthHandleService: auth_handlers_service,
			AuthMiddleService: auth_middlewares_service,
			DataService:       data_service,
		},
	}
}

func (s Server) Run(addr string) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/users/login", s.AuthHandleService.Login)
	mux.HandleFunc("/api/users/register", s.AuthHandleService.Register)

	//Todo: mux.HandleFunc("/files/data", DataHandler(GetDataHandler, SaveDataHandler))
	//Todo: mux.HandleFunc("/files/data/hash", GetHashHandler)

	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error(err.Error())
		return ErrFailedStartServer
	}

	return nil
}
