package auth

import (
	"database/sql"
	"net/http"

	"github.com/braginantonev/mhserver/internal/server/handlers"
)

type AuthHandleService interface {
	Login(w http.ResponseWriter, r *http.Request)
	Register(w http.ResponseWriter, r *http.Request)
}

type Config struct {
	DB           *sql.DB
	JWTSignature string
}

type AuthHandler struct {
	Cfg Config
}

func NewAuthHandler(cfg Config) (AuthHandler, error) {
	if cfg.DB == nil {
		return AuthHandler{}, handlers.ErrDBNotInit
	}

	if cfg.JWTSignature == "" {
		return AuthHandler{}, handlers.ErrJWTSigIsEmpty
	}

	return AuthHandler{
		Cfg: cfg,
	}, nil
}
