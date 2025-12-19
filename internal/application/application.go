package application

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/braginantonev/mhserver/internal/application/di"
	appconfig "github.com/braginantonev/mhserver/internal/config/app"
	"github.com/braginantonev/mhserver/internal/repository/database"
	"github.com/braginantonev/mhserver/internal/server"
	"github.com/go-sql-driver/mysql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ApplicationMode int

const (
	AppMode_MainServerOnly ApplicationMode = iota
	AppMode_SubServersOnly
	AppMode_AllServers

	DATABASE_NAME string = "mhserver"
)

const CONFIG_PATH string = "/usr/share/mhserver/mhserver.conf"

func NewApplicationConfig() appconfig.ApplicationConfig {
	var cfg appconfig.ApplicationConfig

	if _, err := toml.DecodeFile(CONFIG_PATH, &cfg); err != nil {
		panic(fmt.Errorf("%s\n%s", err.Error(), ErrConfigurationNotFound.Error()))
	}

	slog.Info("Configuration loaded.")
	slog.Info(fmt.Sprintf("Available server ram: %s", cfg.AvailableRAM))
	slog.Info(fmt.Sprintf("Server will be started at %s:%s", cfg.SubServers["main"].IP, cfg.SubServers["main"].Port))
	slog.Info(fmt.Sprintf("Server configured to use \"mhserver/%s\" database", DATABASE_NAME))
	slog.Info(fmt.Sprintf("Server workspace path = %s", cfg.WorkspacePath))

	return cfg
}

type Application struct {
	cfg appconfig.ApplicationConfig
	db  *sql.DB
}

func NewApplication() *Application {
	return &Application{
		cfg: NewApplicationConfig(),
	}
}

func (app *Application) InitDB() (err error) {
	if app.db != nil {
		return nil
	}

	app.db, err = database.OpenDB(mysql.Config{
		User:                 "mhserver",
		Passwd:               app.cfg.DB_Pass,
		Net:                  "tcp",
		Addr:                 "127.0.0.1:3306",
		DBName:               "mhs_main",
		AllowNativePasswords: true,
	})
	return
}

func (app *Application) runMain() error {
	if !app.cfg.SubServers["main"].Enabled {
		slog.Warn("main server is disabled. Use -S to use subservers only!")
		return nil
	}

	connections := make(map[string]*grpc.ClientConn)
	user_catalogs := make([]string, 0, len(app.cfg.SubServers)-1)

	//* Sub servers connections
	for name, subserver := range app.cfg.SubServers {
		if !subserver.Enabled || name == "main" {
			if name != "main" {
				slog.Warn("Subserver not enabled. Skip connection.", slog.String("subserver", name))
			}
			continue
		}

		address := fmt.Sprintf("%s:%s", subserver.IP, subserver.Port)

		conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return err
		}

		slog.Info("Create subserver client", slog.String("subserver_name", name), slog.String("address", address))

		connections[name] = conn
		user_catalogs = append(user_catalogs, name)
	}

	data_client := di.GetDataClient(connections["files"])

	auth_service := di.SetupAuthService(app.cfg, app.db, user_catalogs)
	data_service := di.SetupDataService(app.cfg, data_client)

	srv := server.Server{
		AuthService: auth_service,
		DataService: data_service,
	}

	return srv.Serve(app.cfg.SubServers["main"].IP, app.cfg.SubServers["main"].Port)
}

func (app *Application) runSubserver(ctx context.Context, wait bool) error {
	grpc_server := grpc.NewServer()
	var grpc_ip, grpc_port string

	wg := sync.WaitGroup{}

	for name, subserver := range app.cfg.SubServers {
		if !subserver.Enabled || name == "main" {
			if name != "main" {
				slog.Warn("Subserver not enabled. Skip initialization.", slog.String("subserver", name))
			}
			continue
		}

		// Set ip and port for grpc server
		grpc_ip, grpc_port = subserver.IP, subserver.Port

		di.RegisterServer[name](ctx, grpc_server, app.cfg)
		slog.InfoContext(ctx, "Register grpc service", slog.String("service_name", name))
	}

	wg.Add(1)
	go func(ip, port string) {
		defer wg.Done()

		addr := fmt.Sprintf("%s:%s", ip, port)
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			slog.Error("error listen grpc", slog.String("err", err.Error()))
			return
		}

		slog.Info("Serve grpc server", slog.String("address", addr))

		if err := grpc_server.Serve(lis); err != nil {
			slog.Error("error serve grpc server", slog.String("err", err.Error()))
		}
	}(grpc_ip, grpc_port)

	if wait {
		wg.Wait()
	}

	return nil
}

func (app *Application) Run(mode ApplicationMode) error {
	slog.Info("Run application with", slog.Int("mode", int(mode)))

	ctx := context.Background()

	if err := app.InitDB(); err != nil {
		slog.Error("Failed init database", slog.String("error", err.Error()))
		return err
	}

	if mode != AppMode_MainServerOnly {
		err := app.runSubserver(ctx, mode == AppMode_SubServersOnly)
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
