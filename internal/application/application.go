package application

import (
	"fmt"
	"log/slog"
	"os"
	"slices"

	"github.com/BurntSushi/toml"
)

const (
	CONFIGURATION_FILE_NAME       string = "mhserver.conf"
	DEBUG_CONFIGURATION_FILE_NAME string = "mhserver-debug.conf"

	DEFAULT_PORT           string = "8080"
	DEFAULT_WORKSPACE_PATH string = "~/.mhserver/"
)

type Config struct {
	WorkspacePath string
	Address       string
	Port          string
	JWTSignature  string
}

func NewConfig() Config {
	var cfg Config
	conf_file_path := DEBUG_CONFIGURATION_FILE_NAME

	if !slices.Contains(os.Args, "--debug") {
		conf_file_path = DEFAULT_WORKSPACE_PATH + CONFIGURATION_FILE_NAME
	}

	if _, err := toml.DecodeFile(conf_file_path, &cfg); err != nil {
		panic(ERR_CONF_NF)
	}

	slog.Info("Configuration loaded.")
	slog.Info(fmt.Sprintf("Server will be started at %s:%s", cfg.Address, cfg.Port))

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
	//ctx, cancel := context.WithCancel(context.Background())
	//defer cancel()

	//Todo: создание БД пользователей

	//Todo: запуск сервера и его хендлеров

	return nil
}
