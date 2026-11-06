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

func (s emptyServerStream) SetHeader(_ metadata.MD) error {
	return nil
}

func (s emptyServerStream) SendHeader(_ metadata.MD) error {
	return nil
}

func (s emptyServerStream) SetTrailer(_ metadata.MD) {}

func (s emptyServerStream) Context() context.Context {
	return s.ctx
}

func (s emptyServerStream) SendMsg(_ any) error {
	return nil
}

func (s emptyServerStream) RecvMsg(_ any) error {
	return nil
}

func emptyUnaryHandler(ctx context.Context, req any) (any, error) {
	return nil, nil
}

func emptyStreamHandler(srv any, stream grpc.ServerStream) error {
	return nil
}
