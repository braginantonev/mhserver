package di

import (
	"context"

	appconfig "github.com/braginantonev/mhserver/internal/config/application"
	"github.com/braginantonev/mhserver/internal/grpc/data"
	data_pb "github.com/braginantonev/mhserver/proto/data"
	"google.golang.org/grpc"
)

func regDataServer(ctx context.Context, grpc *grpc.Server, app_cfg appconfig.ApplicationConfig, server_cfg appconfig.SubServer) {
	data_pb.RegisterDataServiceServer(grpc, data.NewDataServer(ctx, data.NewDataServerConfig(
		app_cfg.WorkspacePath,
		app_cfg.Memory.WithAllocated(server_cfg.Extra.AllocatedMemory),
	)))
}

func RegisterGrpcServer(ctx context.Context, server_name string, grpc *grpc.Server, app_cfg appconfig.ApplicationConfig) bool {
	switch server_name {
	case "files":
		regDataServer(ctx, grpc, app_cfg, *app_cfg.SubServers[server_name])
	default:
		return false
	}
	return true
}

func GetDataServerClient(conn *grpc.ClientConn) data_pb.DataServiceClient {
	return data_pb.NewDataServiceClient(conn)
}
