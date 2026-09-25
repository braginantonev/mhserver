package interceptors_test

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type emptyServerStream struct {
	ctx context.Context
}

func newEmptyServerStream(ctx context.Context) emptyServerStream {
	return emptyServerStream{
		ctx,
	}
}

func (_ emptyServerStream) SetHeader(_ metadata.MD) error {
	return nil
}

func (_ emptyServerStream) SendHeader(_ metadata.MD) error {
	return nil
}

func (_ emptyServerStream) SetTrailer(_ metadata.MD) {}

func (ess emptyServerStream) Context() context.Context {
	return ess.ctx
}

func (_ emptyServerStream) SendMsg(_ any) error {
	return nil
}

func (_ emptyServerStream) RecvMsg(_ any) error {
	return nil
}

func emptyUnaryHandler(ctx context.Context, req any) (any, error) {
	return nil, nil
}

func emptyStreamHandler(srv any, stream grpc.ServerStream) error {
	return nil
}
