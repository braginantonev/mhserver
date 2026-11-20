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

func (intc *InternalTokenInterceptor) isInternalRequest(md metadata.MD) bool {
	token := md["internal-token"]
	if len(token) < 1 {
		// also with prev return - this is user, not server, request
		return false
	}
	return token[0] == intc.serverToken
}

func (intc *InternalTokenInterceptor) getSenderServiceName(md metadata.MD) (string, bool) {
	sender := md["sender-service"]
	if len(sender) < 1 {
		return "", false
	}
	return sender[0], true
}

func (intc *InternalTokenInterceptor) handle(ctx context.Context) (context.Context, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		// we continue the intc chain, because if req doesn't have metadata - user request
		return ctx, true
	}

	if !intc.isInternalRequest(md) {
		return ctx, true
	}

	service_name, ok := intc.getSenderServiceName(md)
	if !ok {
		slog.ErrorContext(ctx, "sender service not set in internal request")
		return nil, false
	}

	handler_ctx := context.WithValue(ctx, contextkeys.InternalRequest, true)
	handler_ctx = context.WithValue(handler_ctx, contextkeys.SenderService, service_name)

	return handler_ctx, true
}

func (intc *InternalTokenInterceptor) Unary(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (m any, err error) {
	ctx, ok := intc.handle(ctx)
	if !ok {
		return nil, ErrInternal
	}
	return handler(ctx, req)
}

func (intc *InternalTokenInterceptor) Stream(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
	ctx, ok := intc.handle(ss.Context())
	if !ok {
		return ErrInternal
	}
	return handler(srv, NewWrappedStream(ss, ctx))
}
