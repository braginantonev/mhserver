package datahttp

import (
	"net/http"

	"github.com/braginantonev/mhserver/internal/config"
	"golang.org/x/time/rate"
)

type Middleware struct {
	limiter *rate.Limiter
}

func NewMiddleware(req_cfg config.RequestsConfig) Middleware {
	return Middleware{
		limiter: rate.NewLimiter(rate.Every(req_cfg.LimiterInterval), req_cfg.MaxInInterval),
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
