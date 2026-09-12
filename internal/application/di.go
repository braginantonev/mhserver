package application

import (
	"context"

	"github.com/braginantonev/mhserver/internal/config"
	appconfig "github.com/braginantonev/mhserver/internal/config/application"
	"github.com/braginantonev/mhserver/internal/services/data"
	data_pb "github.com/braginantonev/mhserver/proto/gen/data"
	"google.golang.org/grpc"
)

func regDataServer(ctx context.Context, grpc *grpc.Server, app_cfg appconfig.ApplicationConfig, server_cfg appconfig.SubServer) {
	data_pb.RegisterDataServiceServer(grpc, data.NewDataServer(ctx, data.NewDataServerConfig(
		app_cfg.WorkspacePath,
		app_cfg.Memory.WithAllocated(server_cfg.Extra.AllocatedMemory),
	)))
}

func RegisterGrpcServer(ctx context.Context, service config.ServiceName, grpc *grpc.Server, app_cfg appconfig.ApplicationConfig) bool {
	switch service {
	case data.SERVICE_NAME:
		regDataServer(ctx, grpc, app_cfg, *app_cfg.SubServers[data.SERVICE_NAME])
	default:
		return false
	}
	return true
}

/* todo
import (
	"database/sql"

	appconfig "github.com/braginantonev/mhserver/internal/config/application"
	"github.com/braginantonev/mhserver/internal/service/auth"
)

func SetupAuthService(app_cfg appconfig.ApplicationConfig, db *sql.DB) *auth.AuthService {
	available_services := make([]string, 0, len(app_cfg.SubServers))
	for sub := range app_cfg.SubServers {
		available_services = append(available_services, sub)
	}

	return auth.NewAuthService(auth.AuthConfig{
		JWTSignature:  app_cfg.JWTSignature,
		WorkspacePath: app_cfg.WorkspacePath,
		UserCatalogs:  available_services[1:],
	}, db)
}
*/
