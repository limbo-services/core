package limbo

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/limbo-services/trace"
)

func TraceMiddleware() Middleware {
	return &tracerMiddleware{}
}

type tracerMiddleware struct{}

func (t *tracerMiddleware) Apply(desc *grpc.ServiceDesc) {

	for i, m := range desc.Methods {
		desc.Methods[i] = t.wrapMethod(desc, m)
	}

	for i, s := range desc.Streams {
		desc.Streams[i] = t.wrapStream(desc, s)
	}

}

func (t *tracerMiddleware) wrapMethod(srv *grpc.ServiceDesc, desc grpc.MethodDesc) grpc.MethodDesc {
	name := "/" + srv.ServiceName + "/" + desc.MethodName + "/root"
	h := desc.Handler
	desc.Handler = func(srv interface{}, ctx context.Context, dec func(interface{}) error) (out interface{}, err error) {
		span, ctx := trace.New(ctx, name, trace.WithPanicGuard)
		defer func(errPtr *error) {
			if span.Failed && *errPtr == nil {
				*errPtr = grpc.Errorf(codes.Internal, "internal server error")
			}
		}(&err)
		defer span.Close()

		return h(srv, ctx, dec)
	}
	return desc
}

func (t *tracerMiddleware) wrapStream(srv *grpc.ServiceDesc, desc grpc.StreamDesc) grpc.StreamDesc {
	name := "/" + srv.ServiceName + "/" + desc.StreamName + "/root"
	h := desc.Handler
	desc.Handler = func(srv interface{}, stream grpc.ServerStream) (err error) {
		span, ctx := trace.New(stream.Context(), name, trace.WithPanicGuard)
		defer func(errPtr *error) {
			if span.Failed && *errPtr == nil {
				*errPtr = grpc.Errorf(codes.Internal, "internal server error")
			}
		}(&err)
		defer span.Close()

		stream = wrapServerSteamWithContext(stream, ctx)
		return h(srv, stream)
	}
	return desc
}

func wrapServerSteamWithContext(stream grpc.ServerStream, ctx context.Context) grpc.ServerStream {
	return &grpcServerStreamWithContext{ctx, stream}
}

type grpcServerStreamWithContext struct {
	ctx context.Context
	grpc.ServerStream
}

func (s *grpcServerStreamWithContext) Context() context.Context {
	return s.ctx
}
