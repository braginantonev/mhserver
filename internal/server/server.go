package server

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/braginantonev/mhserver/internal/domain"
	"github.com/gorilla/mux"
)

const (
	// Auth

	LOGIN_ENDPOINT    string = "/api/v1/users/login"
	REGISTER_ENDPOINT string = "/api/v1/users/register"

	// Data

	CREATE_CONNECTION_ENDPOINT   string = "/api/v1/files/connect"
	SAVE_DATA_ENDPOINT           string = "/api/v1/files/{uuid:[a-z0-9-]{36}}/save"
	GET_DATA_ENDPOINT            string = "/api/v1/files/{uuid:[a-z0-9-]{36}}/get"
	GET_DATA_SUM_ENDPOINT        string = "/api/v1/files/{uuid:[a-z0-9-]{36}}/sum"
	GET_FILES_ENDPOINT           string = "/api/v1/files"
	GET_AVAILABLE_SPACE_ENDPOINT string = "/api/v1/files/space"
	CREATE_DIR_ENDPOINT          string = "/api/v1/files/mkdir"
	REMOVE_DIR_ENDPOINT          string = "/api/v1/files/rmdir"
)

type Server struct {
	AuthService *domain.HttpAuthService
	DataService *domain.HttpDataService
}

func (s *Server) Serve(addr, tls_cert, tls_key string) error {
	r := mux.NewRouter()

	// Auth service
	r.HandleFunc(LOGIN_ENDPOINT, s.AuthService.Handlers.Login).Methods(http.MethodGet)
	r.HandleFunc(REGISTER_ENDPOINT, s.AuthService.Handlers.Register).Methods(http.MethodPost)

	// Data service
	r.HandleFunc(CREATE_CONNECTION_ENDPOINT, s.AuthService.Middlewares.WithAuth(s.DataService.Handler.CreateConnection)).Methods(http.MethodOptions)
	r.HandleFunc(SAVE_DATA_ENDPOINT, s.AuthService.Middlewares.WithAuth(s.DataService.Handler.SaveData)).Methods(http.MethodPost)
	r.HandleFunc(GET_DATA_ENDPOINT, s.AuthService.Middlewares.WithAuth(s.DataService.Handler.GetData)).Methods(http.MethodGet)
	r.HandleFunc(GET_DATA_SUM_ENDPOINT, s.AuthService.Middlewares.WithAuth(s.DataService.Handler.GetSum)).Methods(http.MethodGet)
	r.HandleFunc(GET_FILES_ENDPOINT, s.AuthService.Middlewares.WithAuth(s.DataService.Handler.GetFiles)).Methods(http.MethodGet)
	r.HandleFunc(GET_AVAILABLE_SPACE_ENDPOINT, s.AuthService.Middlewares.WithAuth(s.DataService.Handler.GetAvailableDiskSpace)).Methods(http.MethodGet)
	r.HandleFunc(CREATE_DIR_ENDPOINT, s.AuthService.Middlewares.WithAuth(s.DataService.Handler.CreateDir)).Methods(http.MethodPost)
	r.HandleFunc(REMOVE_DIR_ENDPOINT, s.AuthService.Middlewares.WithAuth(s.DataService.Handler.RemoveDir)).Methods(http.MethodPost)

	r.HandleFunc("/api/v1/ping", func(w http.ResponseWriter, r *http.Request) {}).Methods(http.MethodPost)

	http.Handle("/api/", r)

	http_srv := &http.Server{
		Handler:      r,
		Addr:         addr,
		WriteTimeout: 5 * time.Second,
		ReadTimeout:  5 * time.Second,
	}

	if err := http_srv.ListenAndServeTLS(tls_cert, tls_key); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrUnsafeProtocol
		}

		slog.Error(err.Error())
		return ErrFailedStartServer
	}

	return nil
}
