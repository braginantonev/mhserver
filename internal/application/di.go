package application

import (
	"context"
	"database/sql"
	"errors"

	"github.com/braginantonev/mhserver/internal/config"
	"github.com/braginantonev/mhserver/internal/services"
	"github.com/braginantonev/mhserver/internal/services/auth"
	"github.com/braginantonev/mhserver/internal/services/data"
	auth_pb "github.com/braginantonev/mhserver/proto/gen/auth"
	data_pb "github.com/braginantonev/mhserver/proto/gen/data"
	"google.golang.org/grpc"
)

var ErrServiceNotFound error = errors.New("service not found")

func RegisterGrpcServer(ctx context.Context, grpc *grpc.Server, service services.ServiceName, app_cfg ApplicationConfig, db *sql.DB) error {
	service_config := string(service) + ".conf"

	switch service {
	case data.SERVICE_NAME:
		cfg := data.NewDataServerConfig(
			app_cfg.WorkspacePath,
			app_cfg.Memory,
		)
		if err := config.LoadConfig(cfg.WorkspacePath, service_config, &cfg); err != nil {
			return err
		}
		data_pb.RegisterDataServiceServer(grpc, data.NewDataServer(ctx, cfg))

	case auth.SERVICE_NAME:
		cfg := auth.NewAuthServiceConfig(
			app_cfg.WorkspacePath,
			app_cfg.JWTSignature,
		)
		if err := config.LoadConfig(cfg.WorkspacePath, service_config, &cfg); err != nil {
			return err
		}
		auth_pb.RegisterAuthServiceServer(grpc, auth.NewAuthServer(cfg, db))

	default:
		return ErrServiceNotFound
	}

	return nil
}
