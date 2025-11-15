package application

import (
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/BurntSushi/toml"
	_ "github.com/go-sql-driver/mysql"
)

const CONFIG_PATH string = "/usr/share/mhserver/mhserver.conf"

type (
	Config struct {
		ServerName    string `toml:"server_name"`
		WorkspacePath string `toml:"workspace_path"`
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

	if _, err := toml.DecodeFile(CONFIG_PATH, &cfg); err != nil {
		panic(fmt.Sprintf("%s\n%s", err.Error(), ErrConfigurationNotFound.Error()))
	}

	slog.Info("Configuration loaded.")
	slog.Info(fmt.Sprintf("Server will be started at %s:%s", cfg.SubServers["main"].IP, cfg.SubServers["main"].Port))
	slog.Info(fmt.Sprintf("Server configured to use \"mhserver/%s\" database", cfg.ServerName))
	slog.Info(fmt.Sprintf("Server workspace path = %s", cfg.WorkspacePath))

	return cfg
}

type Application struct {
	Config
	DB *sql.DB
}

func NewApplication() *Application {
	return &Application{
		Config: NewConfig(),
	}
}

func (app *Application) InitDB() error {
	DB, err := sql.Open("mysql", fmt.Sprintf("mhserver:%s@/%s", app.DB_Pass, app.ServerName))
	if err != nil {
		return err
	}

	if err = DB.Ping(); err != nil {
		return err
	}

	app.DB = DB
	return nil
}
