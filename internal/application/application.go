package application

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/BurntSushi/toml"
)

const (
	CONFIGURATION_FILE_NAME       string = "mhserver.conf"
	DEBUG_CONFIGURATION_FILE_NAME string = "mhserver-debug.conf"

	DEFAULT_WORKSPACE_PATH string = "~/.mhserver/"
)

type Config struct {
	WorkspacePath string
	IP            string
	Port          string
	JWTSignature  string
}

func NewConfig() Config {
	var cfg Config
	conf_file_path := DEBUG_CONFIGURATION_FILE_NAME

	if _, err := toml.DecodeFile(conf_file_path, &cfg); err != nil {
		panic(ERR_CONF_NF)
	}

	slog.Info("Configuration loaded.")
	fmt.Println(cfg)
	slog.Info(fmt.Sprintf("Server will be started at %s:%s", cfg.IP, cfg.Port))

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
	mux := http.NewServeMux()

	mux.HandleFunc("/api/users/login", LoginHandler)
	mux.HandleFunc("/api/users/register", RegisterHandler)
	mux.HandleFunc("/files/data", DataHandler(GetDataHandler, SaveDataHandler))
	mux.HandleFunc("/files/data/hash", GetHashHandler)

	err := http.ListenAndServe(fmt.Sprintf("%s:%s", app.IP, app.Port), mux)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to start server: %s", err.Error()))
		return err
	}

	return nil
}
