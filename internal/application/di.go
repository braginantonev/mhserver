package application

import (
	"context"
	"database/sql"

	"github.com/braginantonev/mhserver/internal/config"
	appconfig "github.com/braginantonev/mhserver/internal/config/application"
	"github.com/braginantonev/mhserver/internal/services/auth"
	"github.com/braginantonev/mhserver/internal/services/data"
	auth_pb "github.com/braginantonev/mhserver/proto/gen/auth"
	data_pb "github.com/braginantonev/mhserver/proto/gen/data"
	"google.golang.org/grpc"
)

func RegisterGrpcServer(ctx context.Context, grpc *grpc.Server, service config.ServiceName, app_cfg appconfig.ApplicationConfig, db *sql.DB) bool {
	switch service {
	case data.SERVICE_NAME:
		data_pb.RegisterDataServiceServer(grpc, data.NewDataServer(ctx, data.NewDataServerConfig(
			config.USER_SPACE_DIRECTORY,
			app_cfg.Memory,
		)))
	case auth.SERVICE_NAME:
		auth_pb.RegisterAuthServiceServer(grpc, auth.NewAuthServer(auth.NewAuthConfig(
			app_cfg.JWTSignature,
		), db))
	default:
		return false
	}
	return true
}
