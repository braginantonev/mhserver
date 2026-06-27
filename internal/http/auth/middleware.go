package authhttp

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	authconfig "github.com/braginantonev/mhserver/internal/config/auth"
	"github.com/braginantonev/mhserver/internal/repository/ratelimit"
	"github.com/braginantonev/mhserver/pkg/httpcontextkeys"
	"github.com/golang-jwt/jwt/v5"
)

type Middleware struct {
	cfg     authconfig.AuthMiddlewareConfig
	limiter *ratelimit.Limiter
}

func NewMiddleware(ctx context.Context, cfg authconfig.AuthMiddlewareConfig) Middleware {
	return Middleware{
		cfg:     cfg,
		limiter: ratelimit.NewLimiter(ctx, cfg.Requests.Limit, cfg.Requests.Interval),
	}
}

func (mid Middleware) WithRateLimit(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req_ip := r.Header.Get("X-Forwarded-For")
		allowed, after := mid.limiter.Allow(req_ip)
		if !allowed {
			w.Header().Add("Retry-After", fmt.Sprint(after))
			ErrToManyRequests.Write(w)
			return
		}

		w.Header().Add("X-RateLimit-Remaining", fmt.Sprint(mid.limiter.Remaining(req_ip)))
		w.Header().Add("X-RateLimit-Limit", fmt.Sprint(mid.limiter.Limit()))
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
