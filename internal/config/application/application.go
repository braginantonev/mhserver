// Пакет с конфигом для application и di
package appconfig

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/braginantonev/mhserver/internal/config"
	"github.com/braginantonev/mhserver/version"
	"github.com/pelletier/go-toml/v2"
)

const (
	CONFIG_FILENAME         string = "mhserver.conf"
	DEFAULT_CONFIG_FILENAME string = CONFIG_FILENAME + ".default"
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

	with_default bool
}

func NewApplicationConfig(load_default bool) ApplicationConfig {
	return ApplicationConfig{
		with_default: load_default,
	}
}

func (cfg *ApplicationConfig) Init(config_dir, db_name string) error {
	if cfg.with_default {
		def, err := os.ReadFile(config_dir + DEFAULT_CONFIG_FILENAME)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return err
			}

			// download from github
			resp, err := http.Get(fmt.Sprintf("https://github.com/braginantonev/mhserver/blob/%s/%s", version.Version, DEFAULT_CONFIG_FILENAME))
			if err != nil {
				return err
			}
			defer func() {
				_ = resp.Body.Close()
			}()

			if resp.StatusCode != 200 {
				return fmt.Errorf("default file not found (%d)", resp.StatusCode)
			}

			def, err = io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
		}

		if err := toml.Unmarshal(def, &cfg); err != nil {
			return err
		}
	}

	from_file, err := os.ReadFile(config_dir + CONFIG_FILENAME)
	if err != nil {
		return err
	}

	if err := toml.Unmarshal(from_file, &cfg); err != nil {
		return err
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

	if priority_sum != 0 {
		mem_chunk := cfg.Memory.Allocated / uint64(priority_sum)
		for n, srv := range cfg.SubServers {
			if srv.Enabled && srv.Extra.AllocatedMemory == 0 {
				srv.Extra.AllocatedMemory = mem_chunk * uint64(srv.Extra.Priority)
				slog.Info("Allocate memory for", slog.String("subserver", n), slog.Any("value", srv.Extra.AllocatedMemory))
			}
		}
	}

	// ^ mb that's look like a shit

	return nil
}
