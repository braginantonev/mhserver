package application

import (
	"os"

	"github.com/braginantonev/mhserver/internal/config"
	"github.com/pelletier/go-toml/v2"
)

type Server struct {
	Address string
	Port    int
}

type Service struct {
	Enabled bool
}

type ApplicationConfig struct {
	JWTSignature string
	DBPass       string
	Server       Server
	Memory       config.MemoryConfig
	Services     map[config.ServiceName]Service
}

func NewApplicationConfig() (ApplicationConfig, error) {
	var cfg ApplicationConfig

	default_cfg, err := os.ReadFile(config.DEFAULT_CONFIG_DIRECTORY + "server.conf.default")
	if err != nil {
		return cfg, err
	}

	// set default values
	if err := toml.Unmarshal(default_cfg, &cfg); err != nil {
		return cfg, err
	}

	from_file, err := os.ReadFile(config.CONFIG_DIRECTORY + "server.conf")
	if err != nil {
		return cfg, err
	}

	// override by user config
	if err := toml.Unmarshal(from_file, &cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}
