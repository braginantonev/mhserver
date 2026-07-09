package data

import "github.com/braginantonev/mhserver/internal/config"

const (
	SERVICE_NAME    config.ServiceName = "files"
	BASE_CHUNK_SIZE uint64             = 32 * 1024 // 32 kb
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
