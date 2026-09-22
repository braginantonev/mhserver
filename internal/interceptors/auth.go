package interceptors

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/braginantonev/mhserver/pkg/httpcontextkeys"
	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var (
	ErrMissingMetadata   error = errors.New("missed metadata in req")
	ErrAuthTokenIsMissed error = errors.New("auth token is missed")
	ErrAuthTokenExpired  error = errors.New("auth token is wrong or expired")
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
	if len(authorization) < 1 {
		return nil, ErrAuthTokenIsMissed
	}

	token := strings.TrimPrefix(authorization[0], "Bearer ")

	return jwt.Parse(token, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%s: %s", "wrong token signature", t.Header["alg"])
		}
		return []byte(inc.signature), nil
	})
}

func (inc *AuthInterceptor) Unary(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, ErrMissingMetadata
	}

	parsed, err := inc.parseToken(md["authorization"])
	if err != nil {
		return nil, ErrAuthTokenExpired
	}

	handler_ctx := context.WithValue(ctx, httpcontextkeys.USERNAME, parsed.Claims.(jwt.MapClaims)["name"].(string))

	m, err := handler(handler_ctx, req)
	if err != nil {
		slog.ErrorContext(handler_ctx, "RPC failed", slog.Any("error", err))
	}

	return m, err
}

func (inc *AuthInterceptor) Stream(srv any, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	md, ok := metadata.FromIncomingContext(ss.Context())
	if !ok {
		return ErrMissingMetadata
	}

	parsed, err := inc.parseToken(md["authorization"])
	if err != nil {
		return ErrAuthTokenExpired
	}

	handler_ctx := context.WithValue(ss.Context(), httpcontextkeys.USERNAME, parsed.Claims.(jwt.MapClaims)["name"].(string))

	wrapped_stream := WrappedStream{
		ServerStream: ss,
		ctx:          handler_ctx,
	}

	err = handler(srv, wrapped_stream)
	if err != nil {
		slog.ErrorContext(handler_ctx, "RPC failed", slog.Any("error", err))
	}

	return err
}
