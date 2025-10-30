package auth_middlewares

import "net/http"

type IAuthMiddleware interface {
	WithAuth(handler http.HandlerFunc) http.HandlerFunc
}

type Config struct {
	JWTSignature string
}

type AuthMiddleware struct {
	cfg Config
}

func NewAuthMiddleware(cfg Config) AuthMiddleware {
	return AuthMiddleware{
		cfg: cfg,
	}
}
