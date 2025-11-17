package auth_middlewares

import (
	"context"
	"errors"
	"net/http"

	"github.com/braginantonev/mhserver/pkg/httpcontextkeys"
	"github.com/golang-jwt/jwt/v5"
)

// Extract username from jwt and put him in request context
func (mid Middleware) WithAuth(handler http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			ErrUserNotAuthorized.Write(w)
			return
		}

		if token[:6] == "Bearer" {
			token = token[7:]
		}

		parsed_token, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, ErrBadJWTToken
			}

			return []byte(mid.cfg.JWTSignature), nil
		})
		if err != nil {
			switch {
			case errors.Is(err, jwt.ErrTokenExpired):
				ErrAuthorizationExpired.Write(w)
			case errors.Is(err, jwt.ErrSignatureInvalid):
				ErrJwtSignatureInvalid.Write(w)
			default:
				ErrBadJWTToken.Write(w)
			}
			return
		}

		if claims, ok := parsed_token.Claims.(jwt.MapClaims); ok {
			r = r.WithContext(context.WithValue(context.Background(), httpcontextkeys.USERNAME, claims["name"].(string)))
		} else {
			//Todo: Internal error
			return
		}

		handler.ServeHTTP(w, r)
	})
}
