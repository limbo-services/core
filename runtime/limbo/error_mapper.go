package limbo

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type ErrorMapper interface {
	HandleError(err error) error
}

func ErrorMapperMiddleware(mapper ErrorMapper) Middleware {
	return &errorMapperMiddleware{mapper}
}

type errorMapperMiddleware struct {
	mapper ErrorMapper
}

func (t *errorMapperMiddleware) Apply(desc *grpc.ServiceDesc) {

	for i, m := range desc.Methods {
		desc.Methods[i] = t.wrapMethod(desc, m)
	}

	for i, s := range desc.Streams {
		desc.Streams[i] = t.wrapStream(desc, s)
	}

}

func (t *errorMapperMiddleware) wrapMethod(srv *grpc.ServiceDesc, desc grpc.MethodDesc) grpc.MethodDesc {
	h := desc.Handler
	desc.Handler = func(srv interface{}, ctx context.Context, dec func(interface{}) error) (out interface{}, err error) {
		res, err := h(srv, ctx, dec)
		if err != nil {
			t.mapper.HandleError(err)
			return nil, err
		}
		return res, nil
	}
	return desc
}

func (t *errorMapperMiddleware) wrapStream(srv *grpc.ServiceDesc, desc grpc.StreamDesc) grpc.StreamDesc {
	h := desc.Handler
	desc.Handler = func(srv interface{}, stream grpc.ServerStream) (err error) {
		err = h(srv, stream)
		if err != nil {
			t.mapper.HandleError(err)
			return err
		}
		return nil
	}
	return desc
}
