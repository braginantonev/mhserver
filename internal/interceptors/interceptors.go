package interceptors

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrInternal error = status.Error(codes.Internal, "internal error")
)

type WrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func NewWrappedStream(parent grpc.ServerStream, ctx context.Context) WrappedStream {
	return WrappedStream{
		ServerStream: parent,
		ctx:          ctx,
	}
}

func (s WrappedStream) Context() context.Context {
	return s.ctx
}
