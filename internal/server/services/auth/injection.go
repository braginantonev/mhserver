package auth

import (
	"database/sql"

	"github.com/braginantonev/mhserver/internal/server/services"
	"github.com/braginantonev/mhserver/internal/server/services/auth/middlewares"
)

type Config struct {
	DB           *sql.DB
	JWTSignature string
}

type AuthService struct {
	cfg         Config
	Middlewares middlewares.AuthMiddleware
}

func NewAuthService(cfg Config, mid middlewares.AuthMiddleware) (AuthService, error) {
	if cfg.DB == nil {
		return AuthService{}, services.ErrDBNotInit
	}

	if cfg.JWTSignature == "" {
		return AuthService{}, services.ErrJWTSigIsEmpty
	}

	return AuthService{
		cfg:         cfg,
		Middlewares: mid,
	}, nil
}
