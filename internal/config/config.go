// Пакет с общими структурами, необходимыми для прочих конфигов.
package config

import "time"

type ServiceName string

type LimiterConfig struct {
	Limit    int
	Interval time.Duration
}

type MemoryConfig struct {
	// Total memory which service can be use
	Allocated uint64 `toml:"available_ram"`

	MaxChunkSize uint64 `toml:"max_chunk_size"`
	MinChunkSize uint64 `toml:"min_chunk_size"`
}

func (m MemoryConfig) WithAllocated(value uint64) MemoryConfig {
	m.Allocated = value
	return m
}
