package data

import (
	"errors"
	"path/filepath"

	"github.com/braginantonev/mhserver/internal/config"
)

const (
	SERVICE_NAME   config.ServiceName = "files"
	SEMAPHORE_SIZE int                = 100
)

type DataServiceConfig struct {
	ServiceName   config.ServiceName `toml:"-"`
	WorkspacePath string             // User files path
	Memory        config.MemoryConfig
}

func NewDataServerConfig(workspace_path string, data_memory_cfg config.MemoryConfig) DataServiceConfig {
	return DataServiceConfig{
		ServiceName:   SERVICE_NAME,
		WorkspacePath: workspace_path,
		Memory:        data_memory_cfg,
	}
}

func (cfg *DataServiceConfig) Init() error {
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
