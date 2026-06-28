package di

import (
	"context"
	"time"

	"github.com/braginantonev/mhserver/internal/config"
	authhttp "github.com/braginantonev/mhserver/internal/http/auth"
	datahttp "github.com/braginantonev/mhserver/internal/http/data"
	"github.com/braginantonev/mhserver/internal/service/auth"
	data_pb "github.com/braginantonev/mhserver/proto/data"
)

func SetupAuthTransport(ctx context.Context, auth_service *auth.AuthService) *authhttp.AuthTransport {
	return authhttp.NewAuthTransport(
		authhttp.NewHandler(auth_service),
		authhttp.NewMiddleware(ctx, auth_service, config.LimiterConfig{
			Limit:    10,
			Interval: time.Minute,
		}),
	)
}

func SetupDataTransport(ctx context.Context, client data_pb.DataServiceClient) *datahttp.DataTransport {
	return datahttp.NewDataTransport(
		datahttp.NewHandler(client),
		datahttp.NewMiddleware(ctx, config.LimiterConfig{
			Limit:    100,
			Interval: time.Second,
		}),
	)
}
