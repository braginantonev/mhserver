package application

import (
	"errors"
	"os"

	"github.com/braginantonev/mhserver/internal/config"
	"github.com/braginantonev/mhserver/internal/services"
)

type Server struct {
	config.ServerSocket
}

type Service struct {
	Enabled bool
}

type ApplicationConfig struct {
	JWTSignature string
	DBPass       string

	WorkspacePath string `toml:"-"`
	Server        Server
	RateLimiter   config.LimiterConfig
	Memory        config.MemoryConfig
	Services      map[services.ServiceName]Service
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

	// get jwt and db pass
	signature, ok := os.LookupEnv("JWT_SIGNATURE")
	if !ok {
		return cfg, errors.New("jwt signature env not found!")
	}

	cfg.JWTSignature = signature

	cfg.DBPass, ok = os.LookupEnv("DATABASE_PASSWORD")
	if !ok {
		return cfg, errors.New("database pass env not found")
	}

	return cfg, nil
}
