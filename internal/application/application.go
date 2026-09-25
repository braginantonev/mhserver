package application

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"

	"github.com/braginantonev/mhserver/internal/repository/database"
	"github.com/go-sql-driver/mysql"
	"google.golang.org/grpc"
)

const (
	DATABASE_USER string = "mhserver"
	DATABASE_NAME string = "mhs_main"
)

type Application struct {
	cfg ApplicationConfig
	db  *sql.DB
}

func NewApplication() (*Application, error) {
	cfg, err := NewApplicationConfig()
	if err != nil {
		return nil, err
	}

	slog.Info("Configuration loaded.")
	slog.Info(fmt.Sprintf("Server will be serve at %s on %d port", cfg.Server.Address, cfg.Server.Port))
	slog.Info(fmt.Sprintf("Server configured to use \"%s/%s\" database", DATABASE_USER, DATABASE_NAME))

	db, err := database.OpenDB(mysql.Config{
		User:                 DATABASE_USER,
		Passwd:               cfg.DBPass,
		Net:                  "tcp",
		Addr:                 "127.0.0.1:3306",
		DBName:               DATABASE_NAME,
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

	for name, subserver := range app.cfg.Services {
		if !subserver.Enabled {
			slog.Warn("Subserver not enabled. Skip initialization.", slog.String("subserver", string(name)))
			continue
		}

		if err := RegisterGrpcServer(ctx, grpc_server, name, app.cfg, app.db); err != nil {
			if errors.Is(err, ErrServiceNotFound) {
				slog.Warn("Subserver enabled, but not realized. Please watch for mhserver updates, to use this service.", slog.String("subserver", string(name)))
			} else {
				slog.Error("failed register service server", slog.Any("error", err))
			}
			continue
		}

		slog.Info("Register grpc service", slog.String("service_name", string(name)))
	}

	var addr_format string
	if strings.ContainsRune(app.cfg.Server.Address, ':') {
		addr_format = "[%s]:%d" // ip v6
	} else {
		addr_format = "%s:%d" // ip v4
	}

	addr := fmt.Sprintf(addr_format, app.cfg.Server.Address, app.cfg.Server.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	slog.Info("Serve grpc server", slog.String("address", addr))

	return grpc_server.Serve(lis)
}
