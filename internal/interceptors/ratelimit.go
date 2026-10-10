package interceptors

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/braginantonev/mhserver/internal/repository/ratelimit"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
)

var (
	ErrUnknownPeer  error          = status.Error(codes.PermissionDenied, "unknown peer")
	StatusExhausted *status.Status = status.New(codes.ResourceExhausted, "rate limit exceeded")
)

func allowRequest(ctx context.Context, limiter *ratelimit.Limiter, ip string) error {
	if allowed, retry := limiter.Allow(ip); !allowed {
		st, err := StatusExhausted.WithDetails(&errdetails.RetryInfo{
			RetryDelay: &durationpb.Duration{
				Seconds: retry,
			},
		})
		if err != nil {
			slog.ErrorContext(ctx, "failed add retry info details to rate limit status", slog.Any("error", err))
			return ErrInternal
		}
		return st.Err()
	}
	return nil
}

func addRateLimitInfo(ctx context.Context, peer string, limiter *ratelimit.Limiter) {
	grpc.SetTrailer(ctx, metadata.New(map[string]string{
		"x-ratelimit-limit":     fmt.Sprint(limiter.Limit()),
		"x-ratelimit-remaining": fmt.Sprint(limiter.Remaining(peer)),
		"x-ratelimit-reset":     fmt.Sprint(limiter.ResetIn(peer)),
	}))
}

type RateLimitStream struct {
	grpc.ServerStream
	peer    string
	limiter *ratelimit.Limiter // use interceptor limiter
}

func (w RateLimitStream) RecvMsg(m any) error {
	if err := allowRequest(w.Context(), w.limiter, w.peer); err != nil {
		return err
	}
	return w.ServerStream.RecvMsg(m)
}

func NewRateLimitStream(parent grpc.ServerStream, limiter *ratelimit.Limiter, peerAddr string) RateLimitStream {
	return RateLimitStream{
		ServerStream: parent,
		peer:         peerAddr,
		limiter:      limiter,
	}
}

// Service interceptor add server token to metadata
type RateLimitInterceptor struct {
	unaryLimiter  *ratelimit.Limiter
	streamLimiter *ratelimit.Limiter
}

func NewRateLimitInterceptor(unaryLimiter, streamLimiter *ratelimit.Limiter) RateLimitInterceptor {
	return RateLimitInterceptor{
		unaryLimiter,
		streamLimiter,
	}
}

func (intc *RateLimitInterceptor) Unary(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (m any, err error) {
	peer, ok := peer.FromContext(ctx)
	if !ok {
		return nil, ErrUnknownPeer
	}

	defer addRateLimitInfo(ctx, peer.Addr.String(), intc.unaryLimiter)

	if err = allowRequest(ctx, intc.unaryLimiter, peer.Addr.String()); err != nil {
		return nil, err
	}

	if m, err = handler(ctx, req); err != nil {
		slog.ErrorContext(ctx, "RPC failed", slog.Any("error", err))
	}
	return m, err
}

func (intc *RateLimitInterceptor) Stream(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
	peer, ok := peer.FromContext(ss.Context())
	if !ok {
		return ErrUnknownPeer
	}

	defer addRateLimitInfo(ss.Context(), peer.Addr.String(), intc.streamLimiter)

	if err = handler(srv, NewRateLimitStream(ss, intc.streamLimiter, peer.Addr.String())); err != nil {
		slog.ErrorContext(ss.Context(), "RPC failed", slog.Any("error", err))
	}
	return err
}
