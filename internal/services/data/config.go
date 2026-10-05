package data

import (
	"github.com/braginantonev/mhserver/internal/services"
)

const (
	SERVICE_NAME       services.ServiceName = "files"
	SEMAPHORE_SIZE     int                  = 100
	DEFAULT_CHUNK_SIZE uint64               = 1 * 1024 * 1024
)

type DataServiceConfig struct {
	services.ServiceConfig

	WorkspacePath string // User files path
	ChunkSize     uint64
	UserSpaces    []string
}

func NewDataServerConfig(workspace_path string, user_spaces []string) DataServiceConfig {
	return DataServiceConfig{
		ServiceName:   SERVICE_NAME,
		WorkspacePath: workspace_path,
		UserSpaces:    user_spaces,
	}
}
