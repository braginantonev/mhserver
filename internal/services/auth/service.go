package auth

import (
	auth_handlers "github.com/braginantonev/mhserver/internal/services/auth/handlers"
	auth_middlewares "github.com/braginantonev/mhserver/internal/services/auth/middlewares"
)

type AuthService struct {
	Handlers    auth_handlers.AuthHandler
	Middlewares auth_middlewares.AuthMiddleware
}

func NewAuthService(hand auth_handlers.AuthHandler, mid auth_middlewares.AuthMiddleware) *AuthService {
	return &AuthService{
		Handlers:    hand,
		Middlewares: mid,
	}
}
