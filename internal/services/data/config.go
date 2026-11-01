package data

import (
	"github.com/braginantonev/mhserver/internal/config"
)

const (
	SERVICE_NAME       config.ServiceName = "files"
	SEMAPHORE_SIZE     int                = 100
	DEFAULT_CHUNK_SIZE uint64             = 1 * 1024 * 1024
)

type DataServiceConfig struct {
	config.ServiceConfig

	// Simply the uspace dir
	WorkspacePath string
	ChunkSize     uint64
	UserSpaces    []string
}

func NewDataServerConfig(workspace_path string, user_spaces []string) DataServiceConfig {
	return DataServiceConfig{
		ServiceName:   SERVICE_NAME,
		WorkspacePath: workspace_path,
		ChunkSize:     DEFAULT_CHUNK_SIZE,
		UserSpaces:    user_spaces,
	}
}
