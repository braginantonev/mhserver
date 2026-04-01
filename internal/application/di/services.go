package di

import (
	"context"
	"database/sql"
	"time"

	"github.com/braginantonev/mhserver/internal/config"
	appconfig "github.com/braginantonev/mhserver/internal/config/app"
	authconfig "github.com/braginantonev/mhserver/internal/config/auth"
	dataconfig "github.com/braginantonev/mhserver/internal/config/data"
	"github.com/braginantonev/mhserver/internal/domain"
	"github.com/braginantonev/mhserver/internal/grpc/data"
	authhttp "github.com/braginantonev/mhserver/internal/http/auth"
	datahttp "github.com/braginantonev/mhserver/internal/http/data"
	data_pb "github.com/braginantonev/mhserver/proto/data"
	"google.golang.org/grpc"
)

func SetupAuthService(app_cfg appconfig.ApplicationConfig, db *sql.DB, user_catalogs []string) *domain.HttpAuthService {
	handler := authhttp.NewHandler(authconfig.AuthHandlerConfig{
		DB:            db,
		JWTSignature:  app_cfg.JWTSignature,
		WorkspacePath: app_cfg.WorkspacePath,
		UserCatalogs:  user_catalogs,
	})

	middleware := authhttp.NewMiddleware(authconfig.AuthMiddlewareConfig{
		JWTSignature: app_cfg.JWTSignature,
		Requests: config.RequestsConfig{
			MaxInInterval:   100,
			LimiterInterval: time.Second,
		},
	})

	return domain.NewAuthService(handler, middleware)
}

func SetupDataService(client data_pb.DataServiceClient) *domain.HttpDataService {
	return domain.NewDataService(
		datahttp.NewHandler(client),
		datahttp.NewMiddleware(config.RequestsConfig{
			MaxInInterval:   100,
			LimiterInterval: time.Second,
		}),
	)
}

//* GRPC

var (
	RegisterServer = map[string]func(context.Context, *grpc.Server, appconfig.ApplicationConfig){
		"files": RegisterDataServer,
	}
)

func RegisterDataServer(ctx context.Context, grpc *grpc.Server, app_cfg appconfig.ApplicationConfig) {
	data_pb.RegisterDataServiceServer(grpc, data.NewDataServer(ctx, dataconfig.DataServiceConfig{
		WorkspacePath: app_cfg.WorkspacePath,
		Memory:        app_cfg.Memory,
	}))
}

func GetDataServerClient(conn *grpc.ClientConn) data_pb.DataServiceClient {
	return data_pb.NewDataServiceClient(conn)
}
