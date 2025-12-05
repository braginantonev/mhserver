package di

import (
	"context"
	"database/sql"

	"github.com/braginantonev/mhserver/internal/configs"
	auth_service "github.com/braginantonev/mhserver/internal/server/services/auth"
	auth_handlers "github.com/braginantonev/mhserver/internal/server/services/auth/handlers"
	auth_middlewares "github.com/braginantonev/mhserver/internal/server/services/auth/middlewares"
	data_service "github.com/braginantonev/mhserver/internal/server/services/data"
	data_handlers "github.com/braginantonev/mhserver/internal/server/services/data/handlers"
	"github.com/braginantonev/mhserver/pkg/data"
	data_pb "github.com/braginantonev/mhserver/proto/data"
	"google.golang.org/grpc"
)

func SetupAuthService(app_cfg configs.ApplicationConfig, db *sql.DB, user_catalogs []string) *auth_service.AuthService {
	handler := auth_handlers.NewAuthHandler(auth_handlers.Config{
		DB:            db,
		JWTSignature:  app_cfg.JWTSignature,
		WorkspacePath: app_cfg.WorkspacePath,
		UserCatalogs:  user_catalogs,
	})

	middleware := auth_middlewares.NewAuthMiddleware(auth_middlewares.Config{
		JWTSignature: app_cfg.JWTSignature,
	})

	return auth_service.NewAuthService(handler, middleware)
}

func SetupDataService(app_cfg configs.ApplicationConfig, client data_pb.DataServiceClient) *data_service.DataService {
	return data_service.NewDataService(data_handlers.NewDataHandler(data_handlers.Config{
		DataConfig:        data.NewDataServerConfig(app_cfg.WorkspacePath, 50), //Todo: Change const chunk size to app.ChunkSize
		MaxRequestsCount:  100,                                                 //Todo: Change const value to app.MaxRequestsCount
		DataServiceClient: client,
	}))
}

//* GRPC

var (
	ServiceRegisterFunc = map[string]func(context.Context, *grpc.Server, configs.ApplicationConfig){
		"files": RegisterDataServer,
	}
)

func RegisterDataServer(ctx context.Context, grpc *grpc.Server, app_cfg configs.ApplicationConfig) {
	data_pb.RegisterDataServiceServer(grpc, data.NewDataServer(ctx, data.Config{
		WorkspacePath: app_cfg.WorkspacePath,
		ChunkSize:     50, //Todo: Change const chunk size to app.ChunkSize
	}))
}

func GetDataServerClient(conn *grpc.ClientConn) data_pb.DataServiceClient {
	return data_pb.NewDataServiceClient(conn)
}

/*func setupGrpcService[S GrpcConstraint](cfg Config) GrpcClientsConstraint {

}*/
