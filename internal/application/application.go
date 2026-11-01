package application

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"sync"

	"github.com/braginantonev/mhserver/internal/config"
	"github.com/braginantonev/mhserver/internal/repository/database"
	"github.com/go-sql-driver/mysql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
	authInterceptor := app.getAuthInterceptor()
	internalTokenInterceptor := app.getInternalTokenInterceptor()
	ratelimitInterceptor := app.getRateLimitInterceptor(ctx)

	grpc_server := grpc.NewServer(
		grpc.MaxRecvMsgSize(int(10*1024*1024)), // additional bytes to avoid memory leak, when max chunk size is very small
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
	dependencies := make(map[config.ServiceName]*grpc.ClientConn)

	deps_warn := sync.OnceFunc(func() {
		slog.WarnContext(ctx, "Dependencies init found!\nPlease ensure your all dependency servers placed in one local net.\nIt's dangerous to place dependencies servers in global net, because clients for him does not provide TLS")
	})

	for name, s := range services {
		if !s.Enabled {
			slog.WarnContext(ctx, "Service not enabled. Skip initialization", slog.String("service", string(name)))
			continue
		}

		enabled = append(enabled, name)

		for dep, socket := range s.Dependencies {
			deps_warn()

			if _, ok := dependencies[dep]; ok {
				continue // dependency connection already initialized
			}

			client, err := grpc.NewClient(socket.Compile(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				slog.ErrorContext(ctx, "Failed create client for dependency", slog.String("dependency", string(dep)), slog.Any("error", err))
				return fmt.Errorf("failed init dependencies")
			}

			dependencies[dep] = client
		}
	}

	// register auth service, because auth config not provided in service configs
	if err := app.registerGrpcServer(ctx, grpc_server, authServiceName(), nil); err != nil {
		slog.ErrorContext(ctx, "Failed register service in server", slog.String("service", "auth"), slog.Any("error", err))
		return fmt.Errorf("failed init services")
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
