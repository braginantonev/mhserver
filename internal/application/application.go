package application

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/BurntSushi/toml"
	_ "github.com/go-sql-driver/mysql"
)

var (
	DB           *sql.DB
	JWTSignature string
)

type (
	Config struct {
		ServerName    string `toml:"server_name"`
		WorkspacePath string
		JWTSignature  string `toml:"jwt_signature"`
		DB_Pass       string `toml:"db_pass"`
		SubServers    map[string]SubServer
	}

	SubServer struct {
		Enabled  bool
		HostName string
		IP       string
		Port     string
	}
)

func NewConfig() Config {
	var cfg Config

	workspacePath, loaded := os.LookupEnv("WORKSPACE_PATH")
	if !loaded {
		panic(fmt.Sprintf("WORKSPACE_PATH %s", ErrEnvironmentNotFound.Error()))
	}
	cfg.WorkspacePath = workspacePath

	config_path, loaded := os.LookupEnv("CONFIG_PATH")
	if !loaded {
		panic(fmt.Sprintf("CONFIG_PATH %s", ErrEnvironmentNotFound.Error()))
	}

	if _, err := toml.DecodeFile(config_path, &cfg); err != nil {
		panic(fmt.Sprintf("%s\n%s", err.Error(), ErrConfigurationNotFound.Error()))
	}

	JWTSignature = cfg.JWTSignature

	slog.Info("Configuration loaded.")
	slog.Info(fmt.Sprintf("Server will be started at %s:%s", cfg.SubServers["main"].IP, cfg.SubServers["main"].Port))
	slog.Info(fmt.Sprintf("Server configured to use \"mhserver/%s\" database", cfg.ServerName))
	slog.Info(fmt.Sprintf("Server workspace path = %s", cfg.WorkspacePath))

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
	var err error

	DB, err = sql.Open("mysql", fmt.Sprintf("mhserver:%s@/%s", app.DB_Pass, app.ServerName))
	if err != nil {
		return err
	}

	if err = DB.Ping(); err != nil {
		return err
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/api/users/login", LoginHandler)
	mux.HandleFunc("/api/users/register", RegisterHandler)
	mux.HandleFunc("/files/data", DataHandler(GetDataHandler, SaveDataHandler))
	mux.HandleFunc("/files/data/hash", GetHashHandler)

	if err = http.ListenAndServe(fmt.Sprintf("%s:%s", app.SubServers["main"].IP, app.SubServers["main"].Port), mux); err != nil {
		return ErrFailedStartServer
	}

	return nil
}
