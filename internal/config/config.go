// Пакет с общими структурами, необходимыми для прочих конфигов.
package config

import (
	"os"
	"time"

	"github.com/pelletier/go-toml/v2"
)

const (
	CONFIG_DIRECTORY         string = "config"
	DEFAULT_CONFIG_DIRECTORY string = "config/default"
	USER_SPACE_DIRECTORY     string = "uspace"
	LOGS_DIRECTORY           string = "logs"
)

type ServerSocket struct {
	Address string
	Port    int16
}

type LimiterConfig struct {
	Enabled  bool
	Limit    int
	Interval time.Duration
}

type MemoryConfig struct {
	MaxChunkSize uint64 `toml:"max_chunk_size"`
	MinChunkSize uint64 `toml:"min_chunk_size"`
}

func InitFromFile[T any](file string, dest *T) error {
	from_file, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	// override by user config
	if err := toml.Unmarshal(from_file, dest); err != nil {
		return err
	}

	return nil
}
