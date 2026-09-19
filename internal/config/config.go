// Пакет с общими структурами, необходимыми для прочих конфигов.
package config

import "time"

const (
	DATABASE_USER string = "mhserver"
	DATABASE_NAME string = "mhs_main"

	WORKSPACE_PATH           string = "/opt/mhserver/"
	CONFIG_DIRECTORY         string = WORKSPACE_PATH + "config/"
	DEFAULT_CONFIG_DIRECTORY string = CONFIG_DIRECTORY + "default/"
	USER_SPACE_DIRECTORY     string = WORKSPACE_PATH + "uspace/"
)

type ServiceName string

type LimiterConfig struct {
	Enabled  bool
	Limit    int
	Interval time.Duration
}

type MemoryConfig struct {
	MaxChunkSize uint64 `toml:"max_chunk_size"`
	MinChunkSize uint64 `toml:"min_chunk_size"`
}
