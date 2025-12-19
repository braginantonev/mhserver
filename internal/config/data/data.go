package dataconfig

type DataServiceConfig struct {
	// User files path
	WorkspacePath string

	//Todo: Upload semaphore
}

func NewDataServerConfig(workspace_path string, chunk_size int) DataServiceConfig {
	return DataServiceConfig{
		WorkspacePath: workspace_path,
	}
}

type DataHandlerConfig struct {
	ServiceConfig    DataServiceConfig
	MaxRequestsCount int
}
