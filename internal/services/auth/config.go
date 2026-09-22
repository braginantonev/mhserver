package auth

import (
	"github.com/braginantonev/mhserver/internal/services"
)

const (
	SERVICE_NAME   services.ServiceName = "auth"
	SEMAPHORE_SIZE int                  = 50
)

type AuthServiceConfig struct {
	services.ServiceConfig

	WorkspacePath string
	JWTSignature  string
}

func NewAuthServiceConfig(workspace_path, jwt_signature string) AuthServiceConfig {
	return AuthServiceConfig{
		ServiceName:   SERVICE_NAME,
		WorkspacePath: workspace_path,
		JWTSignature:  jwt_signature,
	}
}
