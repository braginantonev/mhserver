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
	Handlers    AuthHandler
	Middlewares AuthMiddleware
}

func NewAuthService(hand AuthHandler, mid AuthMiddleware) *HttpAuthService {
	return &HttpAuthService{
		Handlers:    hand,
		Middlewares: mid,
	}
}
