package files

import (
	"github.com/braginantonev/mhserver/internal/config"
)

const (
	SERVICE_NAME       config.ServiceName = "files"
	SEMAPHORE_SIZE     int                = 100
	DEFAULT_CHUNK_SIZE uint64             = 1 * 1024 * 1024
)

type DataServiceConfig struct {
	// Simply the uspace dir
	WorkspacePath string
	ChunkSize     uint64
	UserSpaces    []string

	// Server dirs it's a service dirs. User cannot modify this dirs, but of course he can read it's dir.
	// This is dirs located in the root of uspace!
	//
	// For example we grab music service. User can't add music file from data service (music service provides meta, which data service is doesn't know).
	// But user can download music file (without meta of course), so why not?
	ServiceDirs []string
}

func NewDataServerConfig(workspace_path string, user_spaces, service_dirs []string) DataServiceConfig {
	return DataServiceConfig{
		WorkspacePath: workspace_path,
		ChunkSize:     DEFAULT_CHUNK_SIZE,
		UserSpaces:    user_spaces,
		ServiceDirs:   service_dirs,
	}
}
