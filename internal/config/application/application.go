// Пакет с конфигом для application и di
package appconfig

import (
	"fmt"
	"log/slog"

	"github.com/BurntSushi/toml"
	"github.com/braginantonev/mhserver/internal/config"
)

type SubServer struct {
	Enabled bool
	Address string
	Port    int
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
	Memory        config.MemoryConfig
	SubServers    map[string]*SubServer
}

func NewApplicationConfig(config_path, db_name string) ApplicationConfig {
	var cfg ApplicationConfig

	if _, err := toml.DecodeFile(config_path, &cfg); err != nil {
		panic(fmt.Errorf("%s\n%s", "configuration file have an errors or not found", err.Error()))
	}

	slog.Info("Configuration loaded.")
	slog.Info(fmt.Sprintf("Server allocated memory: %d bytes", cfg.Memory.Allocated))
	slog.Info(fmt.Sprintf("Server will be started at %s:%d", cfg.SubServers["main"].Address, cfg.SubServers["main"].Port))
	slog.Info(fmt.Sprintf("Server configured to use \"mhserver/%s\" database", db_name))
	slog.Info(fmt.Sprintf("Server workspace path = %s", cfg.WorkspacePath))

	// allocate memory for subserver's
	var priority_sum int
	for _, srv := range cfg.SubServers {
		if srv.Enabled && srv.Extra.AllocatedMemory == 0 {
			priority_sum += srv.Extra.Priority
		}
	}

	mem_chunk := cfg.Memory.Allocated / uint64(priority_sum)
	for _, srv := range cfg.SubServers {
		if srv.Enabled && srv.Extra.AllocatedMemory == 0 {
			srv.Extra.AllocatedMemory = mem_chunk * uint64(srv.Extra.Priority)
		}
	}

	// ^ mb that's look like a shit

	return cfg
}
