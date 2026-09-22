package interceptors

import (
	"context"

	"google.golang.org/grpc"
)

type WrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s WrappedStream) Context() context.Context {
	return s.ctx
}
