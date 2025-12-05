package application

import (
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/BurntSushi/toml"
	"github.com/braginantonev/mhserver/internal/configs"
	_ "github.com/go-sql-driver/mysql"
)

type ApplicationMode int

const (
	AppMode_MainServerOnly ApplicationMode = iota
	AppMode_SubServersOnly
	AppMode_AllServers
)

const CONFIG_PATH string = "/usr/share/mhserver/mhserver.conf"

func NewApplicationConfig() configs.ApplicationConfig {
	var cfg configs.ApplicationConfig

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
	configs.ApplicationConfig //Todo: Change to private
	db                        *sql.DB
}

func NewApplication() *Application {
	return &Application{
		ApplicationConfig: NewApplicationConfig(),
	}
}

func (app *Application) InitDB() error {
	db, err := sql.Open("mysql", fmt.Sprintf("mhserver:%s@/%s", app.DB_Pass, app.ServerName))
	if err != nil {
		return err
	}

	if err = db.Ping(); err != nil {
		return err
	}

	app.db = db
	return nil
}

func (app *Application) Run(mode ApplicationMode) error {
	/*ctx := context.Background()

	//* Initialize database
	if err := app.InitDB(); err != nil {
		slog.Error(err.Error())
		return err
	}

	grpc_server := grpc.NewServer()
	grpc_lis, err := net.Listen("tcp", "localhost:8100")
	if err != nil {
		return err
	}

	main_server := server.Server{}

	// Used by auth service, to create user (client) catalogs
	user_catalogs := make([]string, 0, len(app.SubServers))

	for name, _ := range app.SubServers {
		if name == "main" {
			continue
		}

		user_catalogs = append(user_catalogs, name)
	}

	//* Setup local sub servers
	if mode != AppMode_MainServerOnly {
		for name, subserver := range app.SubServers {
			if !subserver.Enabled || name == "main" {
				continue
			}

			if subserver.IP == "localhost" {
				di.RegisterDataServer(ctx, grpc_server)
			}
		}
	}

	//Todo: setup handlers
	//Todo: setup grpc services
	//Todo: setup main server*/

	return nil
}
