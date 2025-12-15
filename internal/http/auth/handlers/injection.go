package auth_handlers

import (
	"database/sql"
	"net/http"
)

type AuthHandler interface {
	Login(w http.ResponseWriter, r *http.Request)
	Register(w http.ResponseWriter, r *http.Request)
}

type Config struct {
	DB           *sql.DB
	JWTSignature string

	WorkspacePath string
	UserCatalogs  []string
}

type Handler struct {
	cfg Config
}

func NewAuthHandler(cfg Config) Handler {
	return Handler{
		cfg: cfg,
	}
}
