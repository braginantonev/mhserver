package auth

import "net/http"

type AuthMiddleService interface {
	WithAuth(handler http.HandlerFunc) http.HandlerFunc
}

type Config struct {
	JWTSignature string
}

type AuthMiddleWare struct {
	Cfg Config
}

func NewAuthMiddleware(config Config) AuthMiddleWare {
	return AuthMiddleWare{
		Cfg: config,
	}
}
