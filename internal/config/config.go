// Пакет с общими структурами, необходимыми для прочих конфигов.
package config

import (
	"os"
	"path/filepath"
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
	Limit    int
	Interval time.Duration
}

func loadConfigFromFile[T any](file string, dest *T) error {
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

func LoadConfig[T any](workspace_path, file string, dest *T) error {
	// init user-override values
	if err := loadConfigFromFile(filepath.Join(workspace_path, CONFIG_DIRECTORY, file), dest); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}
