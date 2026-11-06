package interceptors

import (
	"context"
	"fmt"
	"log/slog"
	"net"

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

func addRateLimitInfo(ctx context.Context, ip string, limiter *ratelimit.Limiter) error {
	return grpc.SetTrailer(ctx, metadata.New(map[string]string{
		"x-ratelimit-limit":     fmt.Sprint(limiter.Limit()),
		"x-ratelimit-remaining": fmt.Sprint(limiter.Remaining(ip)),
		"x-ratelimit-reset":     fmt.Sprint(limiter.ResetIn(ip)),
	}))
}

type RateLimitStream struct {
	grpc.ServerStream
	userIp  string
	limiter *ratelimit.Limiter // use interceptor limiter
}

func (w RateLimitStream) RecvMsg(m any) error {
	if err := allowRequest(w.Context(), w.limiter, w.userIp); err != nil {
		return err
	}
	return w.ServerStream.RecvMsg(m)
}

func NewRateLimitStream(parent grpc.ServerStream, limiter *ratelimit.Limiter, userIp string) RateLimitStream {
	return RateLimitStream{
		ServerStream: parent,
		userIp:       userIp,
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

func (intc *RateLimitInterceptor) getHost(ctx context.Context) (string, error) {
	peer, ok := peer.FromContext(ctx)
	if !ok {
		return "", ErrUnknownPeer
	}

	host, _, err := net.SplitHostPort(peer.Addr.String())
	if err != nil {
		slog.ErrorContext(ctx, "failed split host and port of user peer", slog.Any("error", err))
		return peer.Addr.String(), nil
	}

	return host, nil
}

func (intc *RateLimitInterceptor) Unary(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (m any, err error) {
	host, err := intc.getHost(ctx)
	if err != nil {
		return nil, err
	}

	if err = allowRequest(ctx, intc.unaryLimiter, host); err != nil {
		return nil, err
	}

	defer func() {
		if err = addRateLimitInfo(ctx, host, intc.streamLimiter); err != nil {
			slog.ErrorContext(ctx, "failed add ratelimit info to trailer", slog.Any("error", err))
		}
	}()

	return handler(ctx, req)
}

func (intc *RateLimitInterceptor) Stream(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
	host, err := intc.getHost(ss.Context())
	if err != nil {
		return err
	}

	if err := allowRequest(ss.Context(), intc.streamLimiter, host); err != nil {
		return err
	}

	defer func() {
		if err = addRateLimitInfo(ss.Context(), host, intc.streamLimiter); err != nil {
			slog.ErrorContext(ss.Context(), "failed add ratelimit info to trailer", slog.Any("error", err))
		}
	}()

	return handler(srv, NewRateLimitStream(ss, intc.streamLimiter, host))
}
