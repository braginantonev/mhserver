package authconfig

import (
	"database/sql"

	"github.com/braginantonev/mhserver/internal/config"
)

type AuthHandlerConfig struct {
	DB            *sql.DB
	JWTSignature  string
	WorkspacePath string
	UserCatalogs  []string
}

type AuthMiddlewareConfig struct {
	JWTSignature string
	Requests     config.RequestsConfig
}
