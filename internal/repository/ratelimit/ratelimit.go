package ratelimit

import (
	"context"
	"sync"
	"time"
)

const (
	USER_RATE_LIFETIME     = time.Minute
	LIMITER_CLEAN_DURATION = USER_RATE_LIFETIME / 2
)

type AcceptedRequests struct {
	delete     chan any
	count      int
	next_reset int64
}

func NewAcceptedRequests(reset_duration time.Duration) *AcceptedRequests {
	reqs := &AcceptedRequests{
		delete:     make(chan any),
		next_reset: time.Now().Add(reset_duration).Unix(),
	}
	go reqs.startResetTicker(reset_duration)
	return reqs
}

func (r *AcceptedRequests) reset() {
	r.count = 0
}

func (r *AcceptedRequests) startResetTicker(reset_duration time.Duration) {
	ticker := time.NewTicker(reset_duration)
	defer ticker.Stop()

	for {
		select {
		case <-r.delete:
			return
		case <-ticker.C:
			r.reset()
			r.next_reset = time.Now().Add(reset_duration).Unix()
		}
	}
}

func (r *AcceptedRequests) NextReset() int64 {
	return r.next_reset
}

func (r *AcceptedRequests) Count() int {
	return r.count
}

func (r *AcceptedRequests) Add() {
	r.count += 1
}

func (r *AcceptedRequests) Delete() {
	close(r.delete)
}

type UserRate struct {
	reqs *AcceptedRequests
	exp  int64
}

func NewUserRate(limiter_interval time.Duration) *UserRate {
	return &UserRate{
		reqs: NewAcceptedRequests(limiter_interval),
		exp:  time.Now().Add(USER_RATE_LIFETIME).Unix(),
	}
}

func (u *UserRate) UpdateExpiration() {
	u.exp = time.Now().Add(USER_RATE_LIFETIME).Unix()
}

func (u *UserRate) ResetRequestsAfter() int64 {
	// Сброс запросов в интервале всегда будет больше текущего времени
	return u.reqs.NextReset() - time.Now().Unix()
}

func (u *UserRate) AcceptedRequests() int {
	return u.reqs.Count()
}

func (u *UserRate) RegisterRequest() {
	u.reqs.Add()
	u.UpdateExpiration()
}

func (u *UserRate) Delete() {
	u.reqs.Delete()
}

type Limiter struct {
	mux      *sync.Mutex
	limit    int
	interval time.Duration
	occupied map[string]*UserRate
}

func NewLimiter(ctx context.Context, max_requests int, interval time.Duration) *Limiter {
	limiter := &Limiter{
		mux:      &sync.Mutex{},
		limit:    max_requests,
		interval: interval,
		occupied: make(map[string]*UserRate),
	}
	go limiter.startCleaner(ctx)
	return limiter
}

func (l *Limiter) cleanExpired() {
	now := time.Now().Unix()

	l.mux.Lock()
	for ip, rate := range l.occupied {
		if rate.exp <= now {
			rate.Delete()
			delete(l.occupied, ip)
		}
	}
	l.mux.Unlock()
}

func (l *Limiter) startCleaner(ctx context.Context) {
	ticker := time.NewTicker(LIMITER_CLEAN_DURATION)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			l.cleanExpired()
		}
	}
}

func (l *Limiter) Limit() int {
	return l.limit
}

// Возвращает true, если пользователь не достиг лимита запросов. Если пользователь достиг лимита, возвращается false с временем ожидания в секундах
func (l *Limiter) Allow(ip string) (bool, int64) {
	l.mux.Lock()
	defer l.mux.Unlock()

	rate, ok := l.occupied[ip]
	if !ok {
		rate = NewUserRate(l.interval)
		l.occupied[ip] = rate
	}

	remained := l.limit - rate.AcceptedRequests()
	if remained <= 0 {
		return false, rate.ResetRequestsAfter()
	}

	rate.RegisterRequest()
	return true, 0
}

func (l *Limiter) Remaining(ip string) int {
	l.mux.Lock()
	if rate, ok := l.occupied[ip]; ok {
		return l.limit - rate.AcceptedRequests()
	}
	l.mux.Unlock()

	return l.limit
}
