package ratelimit_test

import (
	"testing"
	"time"

	"github.com/braginantonev/mhserver/internal/repository/ratelimit"
)

const (
	LIMITER_LIMIT    = 10
	LIMITER_INTERVAL = time.Second
)

func TestLimiter(t *testing.T) {
	limiter := ratelimit.NewLimiter(t.Context(), LIMITER_LIMIT, LIMITER_INTERVAL)

	// Use this cases with t.Parallel(), to imitate real server load
	cases := [...]struct {
		name string
		f    func(t *testing.T)
	}{
		{
			name: "all allowed",
			f: func(t *testing.T) {
				t.Parallel()
				for i := range LIMITER_LIMIT * 2 {
					if allowed, _ := limiter.Allow("0.0.0.0"); !allowed {
						t.Fatalf("limiter return not allowed in %d iteration", i+1)
					}
					<-time.After(LIMITER_INTERVAL / 10)
				}
			},
		},
		{
			name: "all allowed parallel",
			f: func(t *testing.T) {
				t.Parallel()
				for i := range 2 {
					err := make(chan int, LIMITER_LIMIT)
					for j := range LIMITER_LIMIT {
						go func(id int) {
							if allowed, _ := limiter.Allow("0.0.0.1"); !allowed {
								err <- id
							}
						}(j + 1)
					}
					close(err)

					for id := range err {
						t.Fatalf("limiter return not allowed in %d iteration, req id - %d", i+1, id)
					}

					<-time.After(LIMITER_INTERVAL)
				}
			},
		},
		{
			name: "one more not allowed",
			f: func(t *testing.T) {
				t.Parallel()
				for range LIMITER_LIMIT {
					limiter.Allow("0.0.0.2")
				}

				if allowed, _ := limiter.Allow("0.0.0.2"); allowed {
					t.Fatalf("limiter return `allowed`, but expected `not allowed`")
				}
			},
		},
	}

	for _, test := range cases {
		t.Run(test.name, test.f)
	}
}
