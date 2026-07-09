package server

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/braginantonev/mhserver/internal/config"
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
	sem           chan any
	AuthTransport *authhttp.AuthTransport
	DataTransport *datahttp.DataTransport
}

func NewServer(memory_cfg config.MemoryConfig) *Server {
	sem_size := (memory_cfg.Allocated * 99 / 100) / (memory_cfg.MaxChunkSize + 1024)
	slog.Info("Set semaphore size", slog.String("subserver", "main"), slog.Int("value", int(sem_size)))

	return &Server{
		sem: make(chan any, sem_size),
	}
}

func (s *Server) WithMainSemaphore(endpoint http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.sem <- struct{}{}
		endpoint.ServeHTTP(w, r)
		<-s.sem
	}
}

func (s *Server) Serve(addr, tls_cert, tls_key string) error {
	r := mux.NewRouter()

	// Auth service
	r.HandleFunc(LOGIN_ENDPOINT, s.WithMainSemaphore(s.AuthTransport.WithRateLimit(s.AuthTransport.Login))).Methods(http.MethodPost)
	r.HandleFunc(REGISTER_ENDPOINT, s.WithMainSemaphore(s.AuthTransport.WithRateLimit(s.AuthTransport.Register))).Methods(http.MethodPost)

	// Data service
	r.HandleFunc(CREATE_CONNECTION_ENDPOINT, s.WithMainSemaphore(s.DataTransport.WithRateLimit(s.AuthTransport.WithAuth(s.DataTransport.CreateConnection)))).Methods(http.MethodPost)
	r.HandleFunc(SAVE_DATA_ENDPOINT, s.WithMainSemaphore(s.DataTransport.WithRateLimit(s.AuthTransport.WithAuth(s.DataTransport.SaveData)))).Methods(http.MethodPost)
	r.HandleFunc(GET_DATA_ENDPOINT, s.WithMainSemaphore(s.DataTransport.WithRateLimit(s.AuthTransport.WithAuth(s.DataTransport.GetData)))).Methods(http.MethodGet)
	r.HandleFunc(GET_DATA_SUM_ENDPOINT, s.WithMainSemaphore(s.DataTransport.WithRateLimit(s.AuthTransport.WithAuth(s.DataTransport.GetSum)))).Methods(http.MethodGet)
	r.HandleFunc(GET_FILES_ENDPOINT, s.WithMainSemaphore(s.DataTransport.WithRateLimit(s.AuthTransport.WithAuth(s.DataTransport.GetFiles)))).Methods(http.MethodGet)
	r.HandleFunc(GET_AVAILABLE_SPACE_ENDPOINT, s.WithMainSemaphore(s.DataTransport.WithRateLimit(s.AuthTransport.WithAuth(s.DataTransport.GetAvailableDiskSpace)))).Methods(http.MethodGet)
	r.HandleFunc(CREATE_DIR_ENDPOINT, s.WithMainSemaphore(s.DataTransport.WithRateLimit(s.AuthTransport.WithAuth(s.DataTransport.CreateDir)))).Methods(http.MethodPost)
	r.HandleFunc(REMOVE_DIR_ENDPOINT, s.WithMainSemaphore(s.DataTransport.WithRateLimit(s.AuthTransport.WithAuth(s.DataTransport.RemoveDir)))).Methods(http.MethodPost)

	ns_limiter := rate.NewLimiter(rate.Every(time.Minute), 10) // limiter for non-service requests

	r.HandleFunc("/api/v1/tools/ping", s.WithMainSemaphore(func(w http.ResponseWriter, r *http.Request) {
		if !ns_limiter.Allow() {
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte("Welcome to MHServer API"))
	})).Methods(http.MethodPost)

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
