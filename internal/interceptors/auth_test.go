package interceptors_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/braginantonev/mhserver/internal/interceptors"
	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	TEST_JWT_SIGNATURE = "test123"
	TEST_USERNAME      = "test user"
)

type TokenType int8

const (
	TokenTypeNormal TokenType = iota
	TokenTypeWrongSignature
	TokenTypeWrongSigningMethod
	TokenTypeExpired
	TokenTypeEmpty
)

func genTestJWT(username string, gen_token_type TokenType) *jwt.Token {
	now := time.Now()

	switch gen_token_type {
	case TokenTypeNormal:
		return jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"name": username,
			"nbf":  now.Unix(),
			"exp":  now.Add(24 * time.Hour).Unix(),
			"iat":  now.Unix(),
		})

	case TokenTypeExpired:
		return jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"name": username,
			"nbf":  now.Add(-10 * time.Second).Unix(),
			"exp":  now.Add(-5 * time.Second).Unix(),
			"iat":  now.Add(-10 * time.Second).Unix(),
		})

	case TokenTypeWrongSigningMethod:
		return jwt.New(jwt.SigningMethodHS384)
	}

	return jwt.New(jwt.SigningMethodHS256)
}

func TestAuthInterceptors(t *testing.T) {
	cases := [...]struct {
		name           string
		jwt_type       TokenType
		expected_error error
	}{
		{
			name:           "expired token",
			jwt_type:       TokenTypeExpired,
			expected_error: interceptors.ErrAuthBadToken,
		},
		{
			name:           "wrong signature",
			jwt_type:       TokenTypeWrongSignature,
			expected_error: interceptors.ErrAuthBadToken,
		},
		{
			name:           "wrong method",
			jwt_type:       TokenTypeWrongSigningMethod,
			expected_error: interceptors.ErrAuthBadToken,
		},
		{
			name:           "normal auth",
			jwt_type:       TokenTypeNormal,
			expected_error: nil,
		},
	}

	inc := interceptors.NewAuthInterceptors(TEST_JWT_SIGNATURE)

	unary_server_info := &grpc.UnaryServerInfo{
		FullMethod: "/test/empty.unary",
	}

	genContextWithToken := func(t *testing.T, jwt_type TokenType) context.Context {
		token := genTestJWT(TEST_USERNAME, jwt_type)

		signature := TEST_JWT_SIGNATURE
		if jwt_type == TokenTypeWrongSignature {
			signature += "garbage"
		}

		token_str, err := token.SignedString([]byte(TEST_JWT_SIGNATURE))
		if err != nil {
			t.Fatalf("failed create signed token: %s", err)
		}

		return metadata.NewIncomingContext(t.Context(), metadata.Pairs("authorization", token_str))
	}

	for _, test := range cases {
		t.Run(fmt.Sprintf("%s unary", test.name), func(t *testing.T) {
			ctx := genContextWithToken(t, test.jwt_type)

			expected, _ := status.FromError(test.expected_error)
			_, err := inc.Unary(ctx, nil, unary_server_info, emptyUnaryHandler)

			st, _ := status.FromError(err)
			if st != expected {
				t.Errorf("\texpected: %s\nbut got: %s", expected, st)
			}
		})
	}

	stream_server_info := &grpc.StreamServerInfo{
		FullMethod: "/test/empty.stream",
	}

	for _, test := range cases {
		t.Run(fmt.Sprintf("%s stream", test.name), func(t *testing.T) {
			ctx := genContextWithToken(t, test.jwt_type)
			wrapped_stream := newEmptyServerStream(ctx)

			expected, _ := status.FromError(test.expected_error)
			err := inc.Stream(nil, wrapped_stream, stream_server_info, emptyStreamHandler)

			st, _ := status.FromError(err)
			if st != expected {
				t.Errorf("\texpected: %s\nbut got: %s", expected, st)
			}
		})
	}
}
