package auth_middlewares

import "net/http"

type AuthMiddleware interface {
	WithAuth(handler http.HandlerFunc) http.HandlerFunc
}

type Config struct {
	JWTSignature string
}

type Middleware struct {
	cfg Config
}

func NewAuthMiddleware(cfg Config) Middleware {
	return Middleware{
		cfg: cfg,
	}
}
