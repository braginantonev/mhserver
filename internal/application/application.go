package application

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"strings"

	appconfig "github.com/braginantonev/mhserver/internal/config/application"
	"github.com/braginantonev/mhserver/internal/repository/database"
	"github.com/go-sql-driver/mysql"
	"google.golang.org/grpc"
)

const (
	DATABASE_NAME    string = "mhserver"
	CONFIG_DIRECTORY string = "/usr/share/mhserver/"
)

type Application struct {
	cfg appconfig.ApplicationConfig
	db  *sql.DB
}

func NewApplication() (*Application, error) {
	var cfg appconfig.ApplicationConfig
	if err := cfg.Init(CONFIG_DIRECTORY, DATABASE_NAME); err != nil {
		return nil, err
	}

	db, err := database.OpenDB(mysql.Config{
		User:                 "mhserver",
		Passwd:               cfg.DB_Pass,
		Net:                  "tcp",
		Addr:                 "127.0.0.1:3306",
		DBName:               "mhs_main",
		AllowNativePasswords: true,
	})
	if err != nil {
		return nil, err
	}

	return &Application{
		cfg: cfg,
		db:  db,
	}, nil
}

func (app *Application) Run(ctx context.Context) error {
	grpc_server := grpc.NewServer(grpc.MaxRecvMsgSize(int(app.cfg.Memory.MaxChunkSize + 1024))) // additional bytes to avoid panic (out of memory), when max chunk size is very small

	for name, subserver := range app.cfg.SubServers {
		if !subserver.Enabled {
			slog.Warn("Subserver not enabled. Skip initialization.", slog.String("subserver", string(name)))
			continue
		}

		if !RegisterGrpcServer(ctx, name, grpc_server, app.cfg) {
			slog.Warn("Subserver enabled, but not realized. Please watch for mhserver updates, to use this service.", slog.String("subserver", string(name)))
			continue
		}

		slog.InfoContext(ctx, "Register grpc service", slog.String("service_name", string(name)))
	}

	var addr_format string
	if strings.ContainsRune(app.cfg.Address, ':') {
		addr_format = "[%s]:%d" // ip v6
	} else {
		addr_format = "%s:%d" // ip v4
	}

	addr := fmt.Sprintf(addr_format, app.cfg.Address, app.cfg.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	slog.Info("Serve grpc server", slog.String("address", addr))

	return grpc_server.Serve(lis)
}
