// Пакет с общими структурами, необходимыми для прочих конфигов.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

const (
	USER_SPACE_DIRECTORY string = "uspace"
	LOGS_DIRECTORY       string = "logs"
)

type Socket struct {
	Address string
	Port    int16
}

// Return compiled socket: address + port
func (s *Socket) Compile() string {
	// IPv6
	if strings.ContainsRune(s.Address, ':') {
		return fmt.Sprintf("[%s]:%d", s.Address, s.Port)
	}

	return fmt.Sprintf("%s:%d", s.Address, s.Port)
}

type LimiterConfig struct {
	Limit    int
	Interval time.Duration
}

type ServiceName string

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
	if err := loadConfigFromFile(filepath.Join(workspace_path, file), dest); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}
