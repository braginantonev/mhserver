package authhttp

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/braginantonev/mhserver/internal/config"
	"github.com/braginantonev/mhserver/internal/repository/ratelimit"
	"github.com/braginantonev/mhserver/internal/service/auth"
	"github.com/braginantonev/mhserver/pkg/httpcontextkeys"
	"github.com/golang-jwt/jwt/v5"
)

type Middleware struct {
	service *auth.AuthService
	limiter *ratelimit.Limiter
}

func NewMiddleware(ctx context.Context, service *auth.AuthService, limiter_cfg config.LimiterConfig) Middleware {
	return Middleware{
		service: service,
		limiter: ratelimit.NewLimiter(ctx, limiter_cfg.Limit, limiter_cfg.Interval),
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

		parsed, err := mid.service.ParseToJWT(token)
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

		claims, ok := parsed.Claims.(jwt.MapClaims)
		if !ok {
			ErrInternal.Write(w)
			return
		}

		r = r.WithContext(context.WithValue(r.Context(), httpcontextkeys.USERNAME, claims["name"].(string)))
		handler.ServeHTTP(w, r)
	})
}
