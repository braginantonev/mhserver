package di

import (
	"context"
	"database/sql"
	"time"

	"github.com/braginantonev/mhserver/internal/config"
	appconfig "github.com/braginantonev/mhserver/internal/config/application"
	authconfig "github.com/braginantonev/mhserver/internal/config/auth"
	"github.com/braginantonev/mhserver/internal/domain"
	"github.com/braginantonev/mhserver/internal/grpc/data"
	authhttp "github.com/braginantonev/mhserver/internal/http/auth"
	datahttp "github.com/braginantonev/mhserver/internal/http/data"
	data_pb "github.com/braginantonev/mhserver/proto/data"
	"google.golang.org/grpc"
)

func SetupAuthService(ctx context.Context, app_cfg appconfig.ApplicationConfig, db *sql.DB) *domain.HttpAuthService {
	user_catalogs := make([]string, 0, len(app_cfg.SubServers))
	for sub := range app_cfg.SubServers {
		user_catalogs = append(user_catalogs, sub)
	}

	handler := authhttp.NewHandler(authconfig.AuthHandlerConfig{
		DB:            db,
		JWTSignature:  app_cfg.JWTSignature,
		WorkspacePath: app_cfg.WorkspacePath,
		UserCatalogs:  user_catalogs[1:], // remove main subserver
	})

	middleware := authhttp.NewMiddleware(ctx, authconfig.AuthMiddlewareConfig{
		JWTSignature: app_cfg.JWTSignature,
		Requests: config.LimiterConfig{
			Limit:    10,
			Interval: time.Minute,
		},
	})

	return domain.NewAuthService(handler, middleware)
}

func SetupDataService(ctx context.Context, client data_pb.DataServiceClient) *domain.HttpDataService {
	return domain.NewDataService(
		datahttp.NewHandler(client),
		datahttp.NewMiddleware(ctx, config.LimiterConfig{
			Limit:    100,
			Interval: time.Second,
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
	data_pb.RegisterDataServiceServer(grpc, data.NewDataServer(ctx, data.NewDataServerConfig(
		app_cfg.WorkspacePath,
		app_cfg.Memory,
	)))
}

func GetDataServerClient(conn *grpc.ClientConn) data_pb.DataServiceClient {
	return data_pb.NewDataServiceClient(conn)
}
