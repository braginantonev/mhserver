package dataconfig

const (
	BASE_CHUNK_SIZE            uint64 = 32 * 1024 // 32 kb
	STANDARD_MAX_SAVE_REQUESTS int    = 125
)

type DataMemoryConfig struct {
	AvailableRAM uint64 `toml:"available_ram"` // Total memory which service can be use
	MaxChunkSize uint64 `toml:"max_chunk_size"`
	MinChunkSize uint64 `toml:"min_chunk_size"`
}

type DataServiceConfig struct {
	WorkspacePath string // User files path
	Memory        DataMemoryConfig
}

func NewDataServerConfig(workspace_path string, data_memory_cfg DataMemoryConfig) DataServiceConfig {
	return DataServiceConfig{
		WorkspacePath: workspace_path,
		Memory:        data_memory_cfg,
	}
}
