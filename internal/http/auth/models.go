package authhttp

import "net/http"

type AuthHandler interface {
	Login(w http.ResponseWriter, r *http.Request)
	Register(w http.ResponseWriter, r *http.Request)
}

type AuthMiddleware interface {
	WithAuth(handler http.HandlerFunc) http.HandlerFunc
	WithRateLimit(http.HandlerFunc) http.HandlerFunc
}

type AuthTransport struct {
	AuthHandler
	AuthMiddleware
}

func NewAuthTransport(handler AuthHandler, middleware AuthMiddleware) *AuthTransport {
	return &AuthTransport{
		AuthHandler:    handler,
		AuthMiddleware: middleware,
	}
}
