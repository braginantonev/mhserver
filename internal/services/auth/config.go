package auth

import (
	"errors"
	"path/filepath"

	"github.com/braginantonev/mhserver/internal/config"
)

const (
	SERVICE_NAME   config.ServiceName = "auth"
	SEMAPHORE_SIZE int                = 50
)

type AuthServiceConfig struct {
	ServiceName   config.ServiceName
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

func (cfg *AuthServiceConfig) Init() error {
	// init default values
	if err := config.InitFromFile(filepath.Join(cfg.WorkspacePath, config.DEFAULT_CONFIG_DIRECTORY), &cfg); err != nil {
		return errors.New("failed init default files service config")
	}

	// init user-override values
	if err := config.InitFromFile(filepath.Join(cfg.WorkspacePath, config.DEFAULT_CONFIG_DIRECTORY), &cfg); err != nil {
		return errors.New("failed init user-override files service config")
	}

	return nil
}
