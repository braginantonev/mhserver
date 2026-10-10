package application

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/braginantonev/mhserver/internal/config"
	"github.com/braginantonev/mhserver/internal/interceptors"
	"github.com/braginantonev/mhserver/internal/repository/ratelimit"
	"github.com/braginantonev/mhserver/internal/services"
	"github.com/braginantonev/mhserver/internal/services/auth"
	"github.com/braginantonev/mhserver/internal/services/data"
	auth_pb "github.com/braginantonev/mhserver/proto/gen/auth"
	data_pb "github.com/braginantonev/mhserver/proto/gen/data"
	"google.golang.org/grpc"
)

var ErrServiceNotFound error = errors.New("service not found")

func (app *Application) registerGrpcServer(ctx context.Context, grpc *grpc.Server, service services.ServiceName) error {
	service_config := string(service) + ".conf"

	switch service {
	case data.SERVICE_NAME:
		workspace_path := filepath.Join(app.cfg.WorkspacePath, "uspace")

		entries, err := os.ReadDir(workspace_path)
		if err != nil {
			return err
		}

		user_spaces := make([]string, 0, len(entries))
		for _, entry := range entries {
			if _, err = os.Readlink(filepath.Join(workspace_path, entry.Name())); err == nil {
				user_spaces = append(user_spaces, entry.Name())
			}
		}

		slog.InfoContext(ctx, "init", slog.Any("user_spaces", user_spaces))

		cfg := data.NewDataServerConfig(
			workspace_path,
			user_spaces,
		)
		if err := config.LoadConfig(app.cfg.WorkspacePath, service_config, &cfg); err != nil {
			return err
		}
		data_pb.RegisterDataServiceServer(grpc, data.NewDataServer(ctx, cfg))

	case auth.SERVICE_NAME:
		cfg := auth.NewAuthServiceConfig(
			app.cfg.WorkspacePath,
			app.cfg.JWTSignature,
		)
		if err := config.LoadConfig(cfg.WorkspacePath, service_config, &cfg); err != nil {
			return err
		}
		auth_pb.RegisterAuthServiceServer(grpc, auth.NewAuthServer(cfg, app.db))

	default:
		return ErrServiceNotFound
	}

	return nil
}

func (app *Application) getAuthInterceptor() interceptors.AuthInterceptor {
	return interceptors.NewAuthInterceptors(app.cfg.JWTSignature)
}

func (app *Application) getInternalTokenInterceptor() interceptors.InternalTokenInterceptor {
	return interceptors.NewServiceInterceptor(app.cfg.ServerToken)
}

func (app *Application) getRateLimitInterceptor(ctx context.Context) interceptors.RateLimitInterceptor {
	unaryCfg, ok := app.cfg.RateLimiter["unary"]
	if !ok {
		unaryCfg = config.LimiterConfig{Limit: 5, Interval: 1}
	}

	streamCfg, ok := app.cfg.RateLimiter["stream"]
	if !ok {
		streamCfg = config.LimiterConfig{Limit: 100, Interval: 5}
	}

	return interceptors.NewRateLimitInterceptor(
		ratelimit.NewLimiter(ctx, unaryCfg.Limit, unaryCfg.Interval),   // unary limiter
		ratelimit.NewLimiter(ctx, streamCfg.Limit, streamCfg.Interval), // stream limiter
	)
}
