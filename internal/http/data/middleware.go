package datahttp

import (
	"context"
	"fmt"
	"net/http"

	"github.com/braginantonev/mhserver/internal/config"
	"github.com/braginantonev/mhserver/internal/repository/ratelimit"
)

type Middleware struct {
	limiter *ratelimit.Limiter
}

func NewMiddleware(req_cfg config.RequestsConfig) Middleware {
	return Middleware{
		limiter: ratelimit.NewLimiter(context.TODO(), req_cfg.MaxInInterval, req_cfg.LimiterInterval),
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

		next.ServeHTTP(w, r)
	}
}
