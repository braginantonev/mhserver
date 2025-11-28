package subservers

import (
	"context"
	"log/slog"
	"net"

	"github.com/braginantonev/mhserver/pkg/data"
	pb "github.com/braginantonev/mhserver/proto/data"
	"google.golang.org/grpc"
)

func EnableDataSubServer(ctx context.Context, addr string, cfg data.Config) {
	grpc_server := grpc.NewServer()
	pb.RegisterDataServiceServer(grpc_server, data.NewDataServer(ctx, cfg))

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		slog.Error("failed listen data sub server", slog.String("error:", err.Error()))
	}

	if err := grpc_server.Serve(lis); err != nil {
		slog.Error("failed serve data sub server", slog.String("error:", err.Error()))
	}
}
