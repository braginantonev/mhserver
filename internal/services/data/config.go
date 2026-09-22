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
}

func NewDataServerConfig(workspace_path string, data_memory_cfg config.MemoryConfig) DataServiceConfig {
	return DataServiceConfig{
		ServiceName:   SERVICE_NAME,
		WorkspacePath: workspace_path,
		Memory:        data_memory_cfg,
	}
}
