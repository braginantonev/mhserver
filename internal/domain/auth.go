package domain

import "net/http"

type AuthHandler interface {
	Login(w http.ResponseWriter, r *http.Request)
	Register(w http.ResponseWriter, r *http.Request)
}

type AuthMiddleware interface {
	WithAuth(handler http.HandlerFunc) http.HandlerFunc
}

type HttpAuthService struct {
	AuthHandler
	AuthMiddleware
}

func NewAuthService(handler AuthHandler, middleware AuthMiddleware) *HttpAuthService {
	return &HttpAuthService{
		AuthHandler:    handler,
		AuthMiddleware: middleware,
	}
}
