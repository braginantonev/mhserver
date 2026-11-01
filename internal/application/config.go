package application

import (
	"errors"
	"os"

	"github.com/braginantonev/mhserver/internal/config"
	"github.com/braginantonev/mhserver/internal/services/data"
)

type Server struct {
	config.Socket
}

type ServiceConfig struct {
	Enabled      bool
	Dependencies map[config.ServiceName]config.Socket
}

type DataConfig struct {
	ServiceConfig
	data.DataServiceConfig
}

type MusicConfig struct {
	ServiceConfig
	// add music config
}

type ImagesConfig struct {
	ServiceConfig
	// add images config
}

type LLMConfig struct {
	ServiceConfig
	// add llm config
}

type ServiceConfigs struct {
	Data         DataConfig
	Music        MusicConfig
	ImagesConfig ImagesConfig
	LLM          LLMConfig
}

func (c *ServiceConfigs) IntoServiceMap() map[config.ServiceName]ServiceConfig {
	return map[config.ServiceName]ServiceConfig{
		data.SERVICE_NAME: c.Data.ServiceConfig,
		"music":           c.Music.ServiceConfig,        // temp name for uncreated service
		"images":          c.ImagesConfig.ServiceConfig, // temp name for uncreated service
		"llm":             c.LLM.ServiceConfig,          // temp name for uncreated service
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

	return cfg, nil
}
