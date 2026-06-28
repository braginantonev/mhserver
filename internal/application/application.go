package application

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"sync"

	appconfig "github.com/braginantonev/mhserver/internal/config/application"
	"github.com/braginantonev/mhserver/internal/di"
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

	DATABASE_NAME    string = "mhserver"
	CONFIG_DIRECTORY string = "/usr/share/mhserver/"
	CONFIG_FILENAME  string = "mhserver.conf"
)

type Application struct {
	cfg appconfig.ApplicationConfig
	db  *sql.DB
}

func NewApplication() *Application {
	return &Application{
		cfg: appconfig.NewApplicationConfig(CONFIG_DIRECTORY+CONFIG_FILENAME, DATABASE_NAME),
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

func (app *Application) runMain(ctx context.Context) error {
	if !app.cfg.SubServers["main"].Enabled {
		slog.Warn("main server is disabled. Use -S to use subservers only!")
		return nil
	}

	connections := make(map[string]*grpc.ClientConn)

	//* Sub servers connections
	for name, subserver := range app.cfg.SubServers {
		if !subserver.Enabled || name == "main" {
			if name != "main" {
				slog.Warn("Subserver not enabled. Skip connection.", slog.String("subserver", name))
			}
			continue
		}

		address := fmt.Sprintf("%s:%d", subserver.Address, subserver.Port)

		conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return err
		}

		slog.Info("Create subserver client", slog.String("subserver_name", name), slog.String("address", address))
		connections[name] = conn
	}

	srv := server.Server{
		AuthService: di.SetupAuthTransport(ctx, di.SetupAuthService(app.cfg, app.db)),
		DataService: di.SetupDataTransport(ctx, di.GetDataServerClient(connections["files"])),
	}

	return srv.Serve(fmt.Sprintf("%s:%d", app.cfg.SubServers["main"].Address, app.cfg.SubServers["main"].Port), CONFIG_DIRECTORY+"ssl/org.crt", CONFIG_DIRECTORY+"ssl/rootCA.key")
}

func (app *Application) runSubserver(ctx context.Context, wait bool) error {
	grpc_server := grpc.NewServer(grpc.MaxRecvMsgSize(int(app.cfg.Memory.MaxChunkSize + 1024))) // additional bytes to avoid panic (out of memory), when max chunk size is very small
	var grpc_address string
	var grpc_port int

	wg := sync.WaitGroup{}

	for name, subserver := range app.cfg.SubServers {
		if !subserver.Enabled || name == "main" {
			if name != "main" {
				slog.Warn("Subserver not enabled. Skip initialization.", slog.String("subserver", name))
			}
			continue
		}

		// Use the last subserver addr and port, for grpc
		grpc_address = subserver.Address
		grpc_port = subserver.Port

		if !di.RegisterGrpcServer(ctx, name, grpc_server, app.cfg) {
			slog.Warn("Subserver enabled, but not realized. Please watch for mhserver updates, to use this service.", slog.String("subserver", name))
			continue
		}

		slog.InfoContext(ctx, "Register grpc service", slog.String("service_name", name))
	}

	wg.Add(1)
	go func(address string, port int) {
		defer wg.Done()

		addr := fmt.Sprintf("%s:%d", address, port)
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			slog.Error("error listen grpc", slog.String("err", err.Error()))
			return
		}

		slog.Info("Serve grpc server", slog.String("address", addr))

		if err := grpc_server.Serve(lis); err != nil {
			slog.Error("error serve grpc server", slog.String("err", err.Error()))
		}
	}(grpc_address, grpc_port)

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
		err := app.runMain(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}
