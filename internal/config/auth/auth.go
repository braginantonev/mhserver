package authconfig

import "database/sql"

type AuthHandlerConfig struct {
	DB           *sql.DB
	JWTSignature string

	WorkspacePath string
	UserCatalogs  []string
}

type AuthMiddlewareConfig struct {
	JWTSignature string
}
