package auth

import (
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

func ParseStringJWT(token, signature string) (*jwt.Token, error) {
	return jwt.Parse(token, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: %s", ErrJwtSignatureInvalid, t.Header["alg"])
		}
		return []byte(signature), nil
	})
}
