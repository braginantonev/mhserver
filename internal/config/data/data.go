package dataconfig

const STANDARD_CHUNK_SIZE = 1024

type DataServiceConfig struct {
	// User files path
	WorkspacePath string

	// Chunk size is a size of one part file, which will be saved
	ChunkSize int

	//Todo: Upload semaphore
}

func NewDataServerConfig(workspace_path string, chunk_size int) DataServiceConfig {
	if chunk_size <= 0 {
		chunk_size = STANDARD_CHUNK_SIZE
	}

	return DataServiceConfig{
		WorkspacePath: workspace_path,
		ChunkSize:     chunk_size,
	}
}

type DataHandlerConfig struct {
	ServiceConfig    DataServiceConfig
	MaxRequestsCount int
}
