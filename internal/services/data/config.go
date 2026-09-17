package data

import "github.com/braginantonev/mhserver/internal/config"

const (
	SERVICE_NAME   config.ServiceName = "files"
	SEMAPHORE_SIZE int                = 100
)

type DataServiceConfig struct {
	ServiceName   config.ServiceName
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
