package server

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	authhttp "github.com/braginantonev/mhserver/internal/http/auth"
	datahttp "github.com/braginantonev/mhserver/internal/http/data"
	"github.com/gorilla/mux"
	"golang.org/x/time/rate"
)

const (
	// Auth

	LOGIN_ENDPOINT    string = "/api/v1/users/login"
	REGISTER_ENDPOINT string = "/api/v1/users/register"

	// Data

	CREATE_CONNECTION_ENDPOINT   string = "/api/v1/files/connect"
	SAVE_DATA_ENDPOINT           string = "/api/v1/files/save"
	GET_DATA_ENDPOINT            string = "/api/v1/files/get"
	GET_DATA_SUM_ENDPOINT        string = "/api/v1/files/sum"
	GET_FILES_ENDPOINT           string = "/api/v1/files"
	GET_AVAILABLE_SPACE_ENDPOINT string = "/api/v1/files/space"
	CREATE_DIR_ENDPOINT          string = "/api/v1/files/mkdir"
	REMOVE_DIR_ENDPOINT          string = "/api/v1/files/rmdir"
)

type Server struct {
	AuthTransport *authhttp.AuthTransport
	DataTransport *datahttp.DataTransport
}

func (s *Server) Serve(addr, tls_cert, tls_key string) error {
	r := mux.NewRouter()

	// Auth service
	r.HandleFunc(LOGIN_ENDPOINT, s.AuthTransport.WithRateLimit(s.AuthTransport.Login)).Methods(http.MethodPost)
	r.HandleFunc(REGISTER_ENDPOINT, s.AuthTransport.WithRateLimit(s.AuthTransport.Register)).Methods(http.MethodPost)

	// Data service
	r.HandleFunc(CREATE_CONNECTION_ENDPOINT, s.DataTransport.WithRateLimit(s.AuthTransport.WithAuth(s.DataTransport.CreateConnection))).Methods(http.MethodPost)
	r.HandleFunc(SAVE_DATA_ENDPOINT, s.DataTransport.WithRateLimit(s.AuthTransport.WithAuth(s.DataTransport.SaveData))).Methods(http.MethodPost)
	r.HandleFunc(GET_DATA_ENDPOINT, s.DataTransport.WithRateLimit(s.AuthTransport.WithAuth(s.DataTransport.GetData))).Methods(http.MethodGet)
	r.HandleFunc(GET_DATA_SUM_ENDPOINT, s.DataTransport.WithRateLimit(s.AuthTransport.WithAuth(s.DataTransport.GetSum))).Methods(http.MethodGet)
	r.HandleFunc(GET_FILES_ENDPOINT, s.DataTransport.WithRateLimit(s.AuthTransport.WithAuth(s.DataTransport.GetFiles))).Methods(http.MethodGet)
	r.HandleFunc(GET_AVAILABLE_SPACE_ENDPOINT, s.DataTransport.WithRateLimit(s.AuthTransport.WithAuth(s.DataTransport.GetAvailableDiskSpace))).Methods(http.MethodGet)
	r.HandleFunc(CREATE_DIR_ENDPOINT, s.DataTransport.WithRateLimit(s.AuthTransport.WithAuth(s.DataTransport.CreateDir))).Methods(http.MethodPost)
	r.HandleFunc(REMOVE_DIR_ENDPOINT, s.DataTransport.WithRateLimit(s.AuthTransport.WithAuth(s.DataTransport.RemoveDir))).Methods(http.MethodPost)

	ns_limiter := rate.NewLimiter(rate.Every(time.Second), 5) // limiter for non-service requests

	r.HandleFunc("/api/v1/tools/ping", func(w http.ResponseWriter, r *http.Request) {
		if !ns_limiter.Allow() {
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte("Welcome to MHServer API"))
	}).Methods(http.MethodPost)

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
