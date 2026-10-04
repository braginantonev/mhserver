package data

import (
	"github.com/braginantonev/mhserver/internal/config"
	"github.com/braginantonev/mhserver/internal/services"
)

const (
	SERVICE_NAME   services.ServiceName = "files"
	SEMAPHORE_SIZE int                  = 100
)

type DataServiceConfig struct {
	services.ServiceConfig

	WorkspacePath string // User files path
	Memory        config.MemoryConfig
	UserSpaces    []string
}

func NewDataServerConfig(workspace_path string, user_spaces []string) DataServiceConfig {
	return DataServiceConfig{
		ServiceName:   SERVICE_NAME,
		WorkspacePath: workspace_path,
		UserSpaces:    user_spaces,
	}
}
