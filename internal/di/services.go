package di

import (
	"context"
	"database/sql"
	"time"

	"github.com/braginantonev/mhserver/internal/config"
	appconfig "github.com/braginantonev/mhserver/internal/config/application"
	"github.com/braginantonev/mhserver/internal/grpc/data"
	authhttp "github.com/braginantonev/mhserver/internal/http/auth"
	datahttp "github.com/braginantonev/mhserver/internal/http/data"
	"github.com/braginantonev/mhserver/internal/service/auth"
	data_pb "github.com/braginantonev/mhserver/proto/data"
	"google.golang.org/grpc"
)

func SetupAuthService(ctx context.Context, app_cfg appconfig.ApplicationConfig, db *sql.DB) *authhttp.AuthTransport {
	user_catalogs := make([]string, 0, len(app_cfg.SubServers))
	for sub := range app_cfg.SubServers {
		user_catalogs = append(user_catalogs, sub)
	}

	service := auth.NewAuthService(auth.AuthConfig{
		JWTSignature:  app_cfg.JWTSignature,
		WorkspacePath: app_cfg.WorkspacePath,
		UserCatalogs:  user_catalogs[1:],
	}, db)

	return authhttp.NewAuthTransport(
		authhttp.NewHandler(service),
		authhttp.NewMiddleware(ctx, service, config.LimiterConfig{
			Limit:    10,
			Interval: time.Minute,
		}),
	)
}

func SetupDataService(ctx context.Context, client data_pb.DataServiceClient) *datahttp.DataTransport {
	return datahttp.NewDataTransport(
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
