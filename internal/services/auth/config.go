package auth

import "github.com/braginantonev/mhserver/internal/config"

const (
	SERVICE_NAME   config.ServiceName = "auth"
	SEMAPHORE_SIZE int                = 50
)

type AuthServiceConfig struct {
	config.ServiceConfig

	JWTSignature string
}

func NewAuthServiceConfig(jwt_signature string) AuthServiceConfig {
	return AuthServiceConfig{
		ServiceName:  SERVICE_NAME,
		JWTSignature: jwt_signature,
	}
}
