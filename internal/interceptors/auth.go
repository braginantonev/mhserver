package interceptors

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/braginantonev/mhserver/pkg/httpcontextkeys"
	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var (
	ErrMissedMetadata    error = status.Error(codes.InvalidArgument, "missed metadata in req")
	ErrAuthTokenIsMissed error = status.Error(codes.InvalidArgument, "auth token is missed")
	ErrAuthBadToken      error = status.Error(codes.Unauthenticated, "auth token is wrong or expired")
)

type AuthInterceptor struct {
	signature string
}

func NewAuthInterceptors(jwt_signature string) AuthInterceptor {
	return AuthInterceptor{
		signature: jwt_signature,
	}
}

func (inc *AuthInterceptor) parseToken(authorization []string) (*jwt.Token, error) {
	token := strings.TrimPrefix(authorization[0], "Bearer ")

	return jwt.Parse(token, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%s: %s", "wrong token signature", t.Header["alg"])
		}
		return []byte(inc.signature), nil
	})
}

func (inc *AuthInterceptor) parseTokenToContext(parent context.Context) (context.Context, error) {
	md, ok := metadata.FromIncomingContext(parent)
	if !ok {
		return nil, ErrMissedMetadata
	}

	authorization := md["authorization"]
	if len(authorization) < 1 {
		return nil, ErrAuthTokenIsMissed
	}

	parsed, err := inc.parseToken(authorization)
	if err != nil {
		return nil, ErrAuthBadToken
	}

	username, ok := parsed.Claims.(jwt.MapClaims)["name"].(string)
	if !ok {
		return nil, ErrAuthBadToken
	}

	return context.WithValue(parent, httpcontextkeys.USERNAME, username), nil
}

func (inc *AuthInterceptor) Unary(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	handler_ctx, err := inc.parseTokenToContext(ctx)
	if err != nil {
		return nil, err
	}

	m, err := handler(handler_ctx, req)
	if err != nil {
		slog.ErrorContext(handler_ctx, "RPC failed", slog.Any("error", err))
	}

	return m, err
}

func (inc *AuthInterceptor) Stream(srv any, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	handler_ctx, err := inc.parseTokenToContext(ss.Context())
	if err != nil {
		return err
	}

	wrapped_stream := WrappedStream{
		ServerStream: ss,
		ctx:          handler_ctx,
	}

	if err = handler(srv, wrapped_stream); err != nil {
		slog.ErrorContext(handler_ctx, "RPC failed", slog.Any("error", err))
	}

	return err
}
