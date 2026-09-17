// Пакет с конфигом для application и di
package appconfig

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/braginantonev/mhserver/internal/config"
	"github.com/pelletier/go-toml/v2"
)

const (
	CONFIG_FILENAME         string = "mhserver.conf"
	DEFAULT_CONFIG_FILENAME string = CONFIG_FILENAME + ".default"
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

func (cfg *ApplicationConfig) Init(config_dir, db_name string) error {
	default_cfg, err := os.ReadFile(config_dir + DEFAULT_CONFIG_FILENAME)
	if err != nil {
		return err
	}

	// set default values
	if err := toml.Unmarshal(default_cfg, &cfg); err != nil {
		return err
	}

	from_file, err := os.ReadFile(config_dir + CONFIG_FILENAME)
	if err != nil {
		return err
	}

	// override by user config
	if err := toml.Unmarshal(from_file, &cfg); err != nil {
		return err
	}

	slog.Info("Configuration loaded.")
	slog.Info(fmt.Sprintf("Server will be serve at %s on %d port", cfg.Server.Address, cfg.Server.Port))
	slog.Info(fmt.Sprintf("Server configured to use \"mhserver/%s\" database", db_name))

	return nil
}
