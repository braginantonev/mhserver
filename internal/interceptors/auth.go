package interceptors

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/braginantonev/mhserver/pkg/contextkeys"
	pb "github.com/braginantonev/mhserver/proto/gen/auth"
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

	NoAuthAvailableMethods = map[string]struct{}{
		pb.AuthService_Register_FullMethodName: {},
		pb.AuthService_Login_FullMethodName:    {},
	}
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

func (inc *AuthInterceptor) addUsernameToContext(parent context.Context) (context.Context, error) {
	md, ok := metadata.FromIncomingContext(parent)
	if !ok {
		return nil, ErrMissedMetadata
	}

	var username string
	// check context on internal request - to not reply check authorization between services
	if isInternal, _ := parent.Value(contextkeys.InternalRequest).(bool); isInternal {
		// we simple get a username from metadata, because it's internal request.
		// in service rpc we create new outgoing context, so previous metadata (from user) must not be used.
		values := md.Get("username")
		if len(values) < 1 {
			return nil, ErrAuthTokenIsMissed
		}
		username = values[0]
	} else {
		authorization := md.Get("authorization")
		if len(authorization) < 1 {
			return nil, ErrAuthTokenIsMissed
		}

		parsed, err := inc.parseToken(authorization)
		if err != nil {
			return nil, ErrAuthBadToken
		}

		username, ok = parsed.Claims.(jwt.MapClaims)["name"].(string)
		if !ok {
			return nil, ErrAuthBadToken
		}
	}

	return context.WithValue(parent, contextkeys.USERNAME, username), nil
}

func (inc *AuthInterceptor) isNoAuthAvailable(target_method string) (ok bool) {
	_, ok = NoAuthAvailableMethods[target_method]
	return
}

func (inc *AuthInterceptor) Unary(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (m any, err error) {
	handler_ctx := ctx

	if !inc.isNoAuthAvailable(info.FullMethod) {
		handler_ctx, err = inc.addUsernameToContext(ctx)
		if err != nil {
			return nil, err
		}
	}

	if m, err = handler(handler_ctx, req); err != nil {
		slog.ErrorContext(handler_ctx, "RPC failed", slog.Any("error", err))
	}

	return m, err
}

func (inc *AuthInterceptor) Stream(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
	handler_ctx := ss.Context()

	if !inc.isNoAuthAvailable(info.FullMethod) {
		handler_ctx, err = inc.addUsernameToContext(ss.Context())
		if err != nil {
			return err
		}
	}

	if err = handler(srv, NewWrappedStream(ss, handler_ctx)); err != nil {
		slog.ErrorContext(handler_ctx, "RPC failed", slog.Any("error", err))
	}

	return err
}

type FakeAuthInterceptor struct {
	username string
}

func NewFakeAuthInterceptor(username string) FakeAuthInterceptor {
	return FakeAuthInterceptor{
		username,
	}
}

func (inc *FakeAuthInterceptor) Unary(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	return handler(context.WithValue(ctx, contextkeys.USERNAME, inc.username), req)
}

func (inc *FakeAuthInterceptor) Stream(srv any, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	return handler(srv, NewWrappedStream(ss, context.WithValue(ss.Context(), contextkeys.USERNAME, inc.username)))
}
