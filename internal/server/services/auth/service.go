package auth

import (
	auth_handlers "github.com/braginantonev/mhserver/internal/server/services/auth/handlers"
	auth_middlewares "github.com/braginantonev/mhserver/internal/server/services/auth/middlewares"
)

type AuthService struct {
	Handlers    auth_handlers.AuthHandler
	Middlewares auth_middlewares.IAuthMiddleware
}

func NewAuthService(hand auth_handlers.AuthHandler, mid auth_middlewares.IAuthMiddleware) *AuthService {
	return &AuthService{
		Handlers:    hand,
		Middlewares: mid,
	}
}
