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

func NewMiddleware(ctx context.Context, req_cfg config.LimiterConfig) Middleware {
	return Middleware{
		limiter: ratelimit.NewLimiter(ctx, req_cfg.Limit, req_cfg.Interval),
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

		w.Header().Add("X-RateLimit-Limit", fmt.Sprint(mid.limiter.Limit()))
		w.Header().Add("X-RateLimit-Remaining", fmt.Sprint(mid.limiter.Remaining(req_ip)))
		w.Header().Add("X-RateLimit-Reset", fmt.Sprint(mid.limiter.ResetIn(req_ip)))
		next.ServeHTTP(w, r)
	}
}
