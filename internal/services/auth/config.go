package auth

import "github.com/braginantonev/mhserver/internal/config"

const (
	SERVICE_NAME config.ServiceName = "auth"
)

type AuthConfig struct {
	ServiceName  config.ServiceName
	JWTSignature string
}

func NewAuthConfig(jwt_signature string) AuthConfig {
	return AuthConfig{
		ServiceName:  SERVICE_NAME,
		JWTSignature: jwt_signature,
	}
}
