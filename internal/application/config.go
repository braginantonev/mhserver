package application

import (
	"errors"
	"fmt"
	"os"

	"github.com/braginantonev/mhserver/internal/config"
	"github.com/braginantonev/mhserver/internal/services/data"
)

type Server struct {
	config.Socket
}

type LocalServiceConfig struct {
	Enabled      bool
	Dependencies map[config.ServiceName]config.Socket
}

type DataConfig struct {
	LocalServiceConfig
	data.DataServiceConfig
}

type MusicConfig struct {
	LocalServiceConfig
	// add music config
}

type ImagesConfig struct {
	LocalServiceConfig
	// add images config
}

type LLMConfig struct {
	LocalServiceConfig
	// add llm config
}

type ServiceConfigs struct {
	Files  DataConfig
	Music  MusicConfig
	Images ImagesConfig
	LLM    LLMConfig
}

func (c *ServiceConfigs) IntoServiceMap() map[config.ServiceName]LocalServiceConfig {
	return map[config.ServiceName]LocalServiceConfig{
		data.SERVICE_NAME: c.Files.LocalServiceConfig,
		"music":           c.Music.LocalServiceConfig,  // temp name for uncreated service
		"images":          c.Images.LocalServiceConfig, // temp name for uncreated service
		"llm":             c.LLM.LocalServiceConfig,    // temp name for uncreated service
	}
}

type ApplicationConfig struct {
	// Secrets
	JWTSignature string
	DBPass       string
	ServerToken  string

	WorkspacePath string
	Server        Server
	RateLimiter   map[string]config.LimiterConfig
	Services      ServiceConfigs
}

func NewApplicationConfig() (ApplicationConfig, error) {
	var cfg ApplicationConfig

	workspace_path, ok := os.LookupEnv("WORKSPACE_PATH")
	if !ok {
		return cfg, errors.New("workspace path env not found")
	}

	if err := config.LoadConfig(workspace_path, "server.conf", &cfg); err != nil {
		return cfg, err
	}

	cfg.WorkspacePath = workspace_path

	signature, ok := os.LookupEnv("JWT_SIGNATURE")
	if !ok {
		return cfg, errors.New("jwt signature env not found")
	}
	cfg.JWTSignature = signature

	cfg.DBPass, ok = os.LookupEnv("DATABASE_PASSWORD")
	if !ok {
		return cfg, errors.New("database pass env not found")
	}

	cfg.ServerToken, ok = os.LookupEnv("SERVER_TOKEN")
	if !ok {
		return cfg, errors.New("server token env not found")
	}

	for k, v := range cfg.Services.IntoServiceMap() {
		fmt.Println(k, v)
	}

	return cfg, nil
}
