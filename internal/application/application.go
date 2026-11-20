package application

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"path/filepath"

	"github.com/braginantonev/mhserver/internal/config"
	"github.com/go-sql-driver/mysql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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

	db_cfg := mysql.Config{
		User:                 DATABASE_USER,
		Passwd:               cfg.Database.Password,
		Net:                  "tcp",
		Addr:                 cfg.Database.Compile(),
		DBName:               DATABASE_NAME,
		AllowNativePasswords: true,
	}

	db, err := sql.Open("mysql", db_cfg.FormatDSN())
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	return &Application{
		cfg: cfg,
		db:  db,
	}, nil
}

func (app *Application) Run(ctx context.Context) error {
	authInterceptor := app.getAuthInterceptor()
	internalTokenInterceptor := app.getInternalTokenInterceptor()
	ratelimitInterceptor := app.getRateLimitInterceptor(ctx)

	creds, err := credentials.NewServerTLSFromFile(filepath.Join(app.cfg.WorkspacePath, "ssl/org.crt"), filepath.Join(app.cfg.WorkspacePath, "ssl/rootCA.key"))
	if err != nil {
		slog.ErrorContext(ctx, "failed create server credentials", slog.Any("error", err))
		return fmt.Errorf("failed init server")
	}

	grpc_server := grpc.NewServer(
		grpc.Creds(creds),
		grpc.MaxRecvMsgSize(int(1024*1024+5*1024)),
		grpc.ChainUnaryInterceptor(
			ratelimitInterceptor.Unary,
			internalTokenInterceptor.Unary,
			authInterceptor.Unary,
		),
		grpc.ChainStreamInterceptor(
			ratelimitInterceptor.Stream,
			internalTokenInterceptor.Stream,
			authInterceptor.Stream,
		),
	)
	defer grpc_server.GracefulStop()

	services := app.cfg.Services.IntoServiceMap()
	enabled := make([]config.ServiceName, 0, len(services))
	conns := make(map[config.Socket]*grpc.ClientConn)
	dependencies := make(map[config.ServiceName]*grpc.ClientConn)

	dependency_creds, err := credentials.NewClientTLSFromFile(filepath.Join(app.cfg.WorkspacePath, "ssl/rootCA.pem"), "")
	if err != nil {
		slog.ErrorContext(ctx, "failed create dependency credentials", slog.Any("error", err))
		return fmt.Errorf("failed init dependencies")
	}

	for name, s := range services {
		if !s.Enabled {
			slog.WarnContext(ctx, "Service not enabled. Skip initialization", slog.String("service", string(name)))
			continue
		}

		enabled = append(enabled, name)

		for dep, socket := range s.Dependencies {
			if _, ok := dependencies[dep]; ok {
				continue // dependency connection already initialized
			}

			if c, ok := conns[socket]; ok {
				dependencies[dep] = c
			} else {
				client, err := grpc.NewClient(socket.Compile(), grpc.WithTransportCredentials(dependency_creds))
				if err != nil {
					slog.ErrorContext(ctx, "Failed create client for dependency", slog.String("dependency", string(dep)), slog.Any("error", err))
					return fmt.Errorf("failed init dependencies")
				}

				conns[socket] = client
				dependencies[dep] = client
			}
		}
	}

	for _, s := range enabled {
		if err := app.registerGrpcServer(ctx, grpc_server, s, dependencies); err != nil {
			slog.ErrorContext(ctx, "Failed register service in server", slog.String("service", string(s)), slog.Any("error", err))
			return fmt.Errorf("failed init services")
		}
	}

	addr := app.cfg.Server.Compile()

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	slog.Info("Serve grpc server", slog.String("address", addr))

	return grpc_server.Serve(lis)
}
