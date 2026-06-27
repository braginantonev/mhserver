// Пакет с общими структурами, необходимыми для прочих конфигов.
package config

import "time"

type ServiceName string

type LimiterConfig struct {
	Limit    int
	Interval time.Duration
}

type MemoryConfig struct {
	AvailableRAM uint64 `toml:"available_ram"` // Total memory which service can be use
	MaxChunkSize uint64 `toml:"max_chunk_size"`
	MinChunkSize uint64 `toml:"min_chunk_size"`
}
