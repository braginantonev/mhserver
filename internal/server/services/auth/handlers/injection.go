package auth_handlers

import (
	"database/sql"
	"net/http"
)

type IAuthHandler interface {
	Login(w http.ResponseWriter, r *http.Request)
	Register(w http.ResponseWriter, r *http.Request)
}

type Config struct {
	DB           *sql.DB
	JWTSignature string
}

type AuthHandler struct {
	cfg Config
}

func NewAuthHandler(cfg Config) AuthHandler {
	return AuthHandler{
		cfg: cfg,
	}
}
