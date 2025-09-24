package app

import (
	"context"
	"log/slog"
	"os"
)

const (
	DEFAULT_PORT           string = "8080"
	DEFAULT_WORKSPACE_PATH string = "~/.mhserver"

	ENV_PORT_NAME           string = "MHSERVER_PORT"
	ENV_JWT_NAME            string = "JWT_SIGNATURE"
	ENV_WORKSPACE_PATH_NAME string = "MHSERVER_WORKSPACE_PATH"
)

type Config struct {
	Port          string
	WorkspacePath string
	JWTSignature  string
}

func NewConfig() Config {
	cfg := Config{}
	var is_env_loaded bool

	cfg.Port, is_env_loaded = os.LookupEnv(ENV_PORT_NAME)
	if !is_env_loaded {
		cfg.Port = DEFAULT_PORT
		slog.Warn(WARN_ENV_NF, slog.String(ENV_PORT_NAME, DEFAULT_PORT))
	}

	cfg.WorkspacePath, is_env_loaded = os.LookupEnv(ENV_WORKSPACE_PATH_NAME)
	if !is_env_loaded {
		cfg.WorkspacePath = DEFAULT_WORKSPACE_PATH
		slog.Warn(WARN_ENV_NF, slog.String(ENV_WORKSPACE_PATH_NAME, DEFAULT_WORKSPACE_PATH))
	}

	cfg.JWTSignature, is_env_loaded = os.LookupEnv(ENV_JWT_NAME)
	if !is_env_loaded {
		slog.Error(ERR_ENV_NF, slog.String("env", ENV_JWT_NAME))
	}

	return cfg
}

type Application struct {
	Config
}

func NewApplication() *Application {
	return &Application{
		Config: NewConfig(),
	}
}

func (app *Application) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//Todo: создание БД пользователей

	//Todo: запуск сервера и его хендлеров
}
