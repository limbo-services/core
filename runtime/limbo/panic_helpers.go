package limbo

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type ErrorHandler interface {
	HandlePanic(r interface{}) error
	HandleError(err error) error
}

func RecoverPanic(errPtr *error, handler ErrorHandler) {
	if r := recover(); r != nil {
		*errPtr = handler.HandlePanic(r)
	} else if *errPtr != nil {
		*errPtr = handler.HandleError(*errPtr)
	}
}

func WrapServerSteamWithContext(stream grpc.ServerStream, ctx context.Context) grpc.ServerStream {
	return &grpcServerStreamWithContext{ctx, stream}
}

type grpcServerStreamWithContext struct {
	ctx context.Context
	grpc.ServerStream
}

func (s *grpcServerStreamWithContext) Context() context.Context {
	return s.ctx
}
