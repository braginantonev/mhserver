package application

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/BurntSushi/toml"
)

const (
	CONFIGURATION_FILE_NAME string = "mhserver.conf"
)

var (
	AuthDB *sql.DB
)

type Config struct {
	IP           string
	Port         string
	JWTSignature string
}

func NewConfig() Config {
	var cfg Config

	workspacePath, loaded := os.LookupEnv("WORKSPACE_PATH")
	if !loaded {
		panic(fmt.Sprintf("WORKSPACE_PATH %s", ERR_ENV_NF.Error()))
	}

	confFilePath := workspacePath + CONFIGURATION_FILE_NAME

	if _, err := toml.DecodeFile(confFilePath, &cfg); err != nil {
		panic(fmt.Sprintf("%s\n%s", err.Error(), ERR_CONF_NF.Error()))
	}

	slog.Info("Configuration loaded.")
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

	if err := http.ListenAndServe(fmt.Sprintf("%s:%s", app.IP, app.Port), mux); err != nil {
		slog.Error(err.Error())
		return ERR_BAD_START_SERVER
	}

	return nil
}
