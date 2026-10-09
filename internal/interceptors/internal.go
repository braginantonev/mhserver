package interceptors

import (
	"context"
	"log/slog"

	"github.com/braginantonev/mhserver/pkg/contextkeys"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// Service interceptor add server token to metadata
type InternalTokenInterceptor struct {
	serverToken string
}

func NewServiceInterceptor(server_token string) InternalTokenInterceptor {
	return InternalTokenInterceptor{
		server_token,
	}
}

func (intc *InternalTokenInterceptor) compareTokenFromContext(ctx context.Context) bool {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		// we not return an error, because if internal server token is missed - this is user, not server, request
		return false
	}

	token := md["internal-token"]
	if len(token) < 1 {
		// also with prev return - this is user, not server, request
		return false
	}

	return token[0] == intc.serverToken
}

func (intc *InternalTokenInterceptor) Unary(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (m any, err error) {
	handlerCtx := ctx
	if intc.compareTokenFromContext(ctx) {
		// we bool value to protect serverToken for logs.
		// using bool val also is protected, because context used only in local machine
		handlerCtx = context.WithValue(ctx, contextkeys.InternalRequest, true)
	}

	if m, err = handler(handlerCtx, req); err != nil {
		slog.ErrorContext(handlerCtx, "RPC failed", slog.Any("error", err))
	}
	return m, err
}

func (intc *InternalTokenInterceptor) Stream(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
	handlerCtx := ss.Context()
	if intc.compareTokenFromContext(ss.Context()) {
		handlerCtx = context.WithValue(ss.Context(), contextkeys.InternalRequest, true) // like in unary
	}

	if err = handler(handlerCtx, ss); err != nil {
		slog.ErrorContext(handlerCtx, "RPC failed", slog.Any("error", err))
	}
	return err
}
