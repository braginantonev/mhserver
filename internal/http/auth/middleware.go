package authhttp

import (
	"context"
	"errors"
	"net/http"

	authconfig "github.com/braginantonev/mhserver/internal/config/auth"
	"github.com/braginantonev/mhserver/pkg/httpcontextkeys"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/time/rate"
)

type Middleware struct {
	cfg     authconfig.AuthMiddlewareConfig
	limiter *rate.Limiter
}

func NewAuthMiddleware(cfg authconfig.AuthMiddlewareConfig) Middleware {
	return Middleware{
		cfg:     cfg,
		limiter: rate.NewLimiter(rate.Every(cfg.Requests.LimiterInterval), cfg.Requests.MaxInInterval),
	}
}

func (mid Middleware) WithRateLimit(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !mid.limiter.Allow() {
			ErrToManyRequests.Write(w)
			return
		}
		next.ServeHTTP(w, r)
	}
}

// Extract username from jwt and put him in request context
func (mid Middleware) WithAuth(handler http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			ErrUserNotAuthorized.Write(w)
			return
		}

		if token[:6] == "Bearer" {
			token = token[7:]
		}

		parsed_token, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, ErrBadJWTToken
			}

			return []byte(mid.cfg.JWTSignature), nil
		})
		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				ErrAuthorizationExpired.Write(w)
			} else if errors.Is(err, jwt.ErrSignatureInvalid) {
				ErrJwtSignatureInvalid.Write(w)
			} else {
				ErrBadJWTToken.Write(w)
			}
			return
		}

		claims, ok := parsed_token.Claims.(jwt.MapClaims)
		if !ok {
			ErrInternal.Write(w)
			return
		}

		r = r.WithContext(context.WithValue(r.Context(), httpcontextkeys.USERNAME, claims["name"].(string)))
		handler.ServeHTTP(w, r)
	})
}
