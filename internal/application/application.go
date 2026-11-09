package application

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"sync"

	"github.com/braginantonev/mhserver/internal/config"
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

	db_cfg := mysql.Config{
		User:                 DATABASE_USER,
		Passwd:               cfg.Database.Password,
		Net:                  "tcp",
		Addr:                 cfg.Database.Socket.Compile(),
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
	conns := make(map[config.Socket]*grpc.ClientConn)
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

			if c, ok := conns[socket]; ok {
				dependencies[dep] = c
			} else {
				client, err := grpc.NewClient(socket.Compile(), grpc.WithTransportCredentials(insecure.NewCredentials()))
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
