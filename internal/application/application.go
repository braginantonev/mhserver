package application

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"

	"github.com/BurntSushi/toml"
	"github.com/braginantonev/mhserver/internal/application/di"
	"github.com/braginantonev/mhserver/internal/configs"
	"github.com/braginantonev/mhserver/internal/server"
	_ "github.com/go-sql-driver/mysql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
	if app.db != nil {
		return nil
	}

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

func (app *Application) runMain() error {
	connections := make(map[string]*grpc.ClientConn)
	user_catalogs := make([]string, 0, len(app.SubServers)-1)

	//* Sub servers connections
	for name, subserver := range app.SubServers {
		if !subserver.Enabled || name == "main" {
			continue
		}

		conn, err := grpc.NewClient(fmt.Sprintf("%s:%s", subserver.IP, subserver.Port), grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return err
		}
		connections[name] = conn
		user_catalogs = append(user_catalogs, name)
	}

	data_client := di.GetDataClient(connections["files"])

	auth_service := di.SetupAuthService(app.ApplicationConfig, app.db, user_catalogs)
	data_service := di.SetupDataService(app.ApplicationConfig, data_client)

	srv := server.NewServer(auth_service, data_service)

	return srv.Serve(app.SubServers["main"].IP, app.SubServers["main"].Port)
}

func (app *Application) runSubserver(ctx context.Context) error {
	grpc_server := grpc.NewServer()
	var g_ip, g_port string

	for name, subserver := range app.SubServers {
		if !subserver.Enabled || name == "main" {
			continue
		}

		// Set ip and port for grpc server
		g_ip, g_port = subserver.IP, subserver.Port

		di.ServiceRegisterFunc[name](ctx, grpc_server, app.ApplicationConfig)
	}

	go func(ctx context.Context, grpc *grpc.Server, ip, port string) {
		lis, err := net.Listen("tcp", fmt.Sprintf("%s:%s", ip, port))
		if err != nil {
			slog.ErrorContext(ctx, "error listen grpc", slog.String("err", err.Error()))
			return
		}

		if err := grpc.Serve(lis); err != nil {
			slog.ErrorContext(ctx, "error serve grpc server", slog.String("err", err.Error()))
		}
	}(ctx, grpc_server, g_ip, g_port)

	return nil
}

func (app *Application) Run(mode ApplicationMode) error {
	ctx := context.Background()

	if err := app.InitDB(); err != nil {
		slog.Error(err.Error())
		return err
	}

	if mode != AppMode_MainServerOnly {
		err := app.runSubserver(ctx)
		if err != nil {
			return err
		}
	}

	if mode != AppMode_SubServersOnly {
		err := app.runMain()
		if err != nil {
			return err
		}
	}

	return nil
}
