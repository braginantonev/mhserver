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

type SubServer struct {
	Enabled bool
	Extra   SubServerExtra
}

type SubServerExtra struct {
	Priority        int
	AllocatedMemory uint64 `toml:"allocated_memory"`
}

type ApplicationConfig struct {
	WorkspacePath string `toml:"workspace_path"`
	JWTSignature  string `toml:"jwt_signature"`
	DB_Pass       string `toml:"db_pass"`
	Address       string
	Port          int
	Memory        config.MemoryConfig
	SubServers    map[config.ServiceName]*SubServer
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
	slog.Info(fmt.Sprintf("Server allocated memory: %d bytes", cfg.Memory.Allocated))
	slog.Info(fmt.Sprintf("Server will be serve at %s on %d port", cfg.Address, cfg.Port))
	slog.Info(fmt.Sprintf("Server configured to use \"mhserver/%s\" database", db_name))
	slog.Info(fmt.Sprintf("Server workspace path = %s", cfg.WorkspacePath))

	// allocate memory for subserver's
	var priority_sum int
	for _, srv := range cfg.SubServers {
		if srv.Enabled && srv.Extra.AllocatedMemory == 0 {
			priority_sum += srv.Extra.Priority
		}
	}

	if priority_sum != 0 {
		mem_chunk := cfg.Memory.Allocated / uint64(priority_sum)
		for name, srv := range cfg.SubServers {
			if srv.Enabled && srv.Extra.AllocatedMemory == 0 {
				srv.Extra.AllocatedMemory = mem_chunk * uint64(srv.Extra.Priority)
				slog.Debug("Allocate memory for", slog.String("subserver", string(name)), slog.Any("value", srv.Extra.AllocatedMemory))
			}
		}
	}

	// ^ mb that's look like a shit

	if err := cfg.checkMemoryIncompatibility(); err != nil {
		return err
	}

	return nil
}
