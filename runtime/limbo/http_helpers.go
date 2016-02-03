package limbo

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	pb "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	"github.com/fd/featherhead/pkg/log"
)

func RenderMessageJSON(w http.ResponseWriter, code int, msg proto.Message) {
	const ContentType = "Content-Type"
	const JSONType = "application/json; charset=utf-8"

	w.Header().Set(ContentType, JSONType)
	w.WriteHeader(code)

	m := jsonpb.Marshaler{}
	err := m.Marshal(w, msg)
	if err != nil {
		log.Logger.Error(err)
	}
}

const metadataHeaderPrefix = "Grpc-Metadata-"

type grpcGatewayFlag struct{}

/*
AnnotateContext adds context information such as metadata from the request.

If there are no metadata headers in the request, then the context returned
will be the same context.
*/
func AnnotateContext(ctx context.Context, req *http.Request) context.Context {
	ctx = context.WithValue(ctx, grpcGatewayFlag{}, true)

	var pairs []string
	for key, vals := range req.Header {
		for _, val := range vals {
			if key == "Authorization" {
				pairs = append(pairs, key, val)
				continue
			}
			if strings.HasPrefix(key, metadataHeaderPrefix) {
				pairs = append(pairs, key[len(metadataHeaderPrefix):], val)
			}
		}
	}

	if len(pairs) == 0 {
		return ctx
	}
	return metadata.NewContext(ctx, metadata.Pairs(pairs...))
}

func FromGateway(ctx context.Context) bool {
	v, ok := ctx.Value(grpcGatewayFlag{}).(bool)
	return v && ok
}

var (
	_ grpc.ServerStream = (*sseServerStream)(nil)
	_ grpc.ServerStream = (*PagedServerStream)(nil)
)

type sseServerStream struct {
	ctx     context.Context
	rw      http.ResponseWriter
	flusher http.Flusher

	mtx         sync.Mutex
	err         error
	headerSent  bool
	messageSent bool
	writeBuffer bytes.Buffer
}

// Context returns the context for this stream.
func (s *sseServerStream) Context() context.Context {
	return s.ctx
}

// SendMsg blocks until it sends m, the stream is done or the stream
// breaks.
// On error, it aborts the stream and returns an RPC status on client
// side. On server side, it simply returns the error to the caller.
// SendMsg is called by generated code.
func (s *sseServerStream) SendMsg(m interface{}) error {
	var (
		messageSent bool
		marshaler   jsonpb.Marshaler
	)

	s.writeBuffer.Reset()
	s.writeBuffer.WriteString("event: message\ndata: ")
	err := marshaler.Marshal(&s.writeBuffer, m.(proto.Message))
	if err != nil {
		return err
	}
	s.writeBuffer.WriteString("\n\n")

	s.mtx.Lock()
	defer s.mtx.Unlock()

	s.messageSent, messageSent = true, s.messageSent

	if s.err != nil {
		return s.err
	}

	if !messageSent {
		s.rw.Header().Set("Content-Type", "text/event-stream")
		s.rw.WriteHeader(200)
	}

	_, err = s.writeBuffer.WriteTo(s.rw)
	s.writeBuffer.Reset()
	if err != nil {
		s.err = err
		return err
	}

	s.flusher.Flush()
	return nil
}

// RecvMsg blocks until it receives a message or the stream is
// done. On client side, it returns io.EOF when the stream is done. On
// any other error, it aborts the streama nd returns an RPC status. On
// server side, it simply returns the error to the caller.
func (s *sseServerStream) RecvMsg(m interface{}) error {
	return io.EOF
}

// SendHeader sends the header metadata. It should not be called
// after SendProto. It fails if called multiple times or if
// called after SendProto.
func (s *sseServerStream) SendHeader(md metadata.MD) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if s.headerSent {
		return errors.New("unable to call SendHeader multiple times")
	}
	if s.messageSent {
		return errors.New("unable to call SendHeader after SendMsg")
	}

	// record call to headerSent
	s.headerSent = true

	// Apply headers
	for k, v := range md {
		for _, vv := range v {
			s.rw.Header().Add(k, vv)
		}
	}

	return nil
}

// SetTrailer sets the trailer metadata which will be sent with the
// RPC status.
func (s *sseServerStream) SetTrailer(md metadata.MD) {
	if md.Len() == 0 {
		return
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()
}

func NewPagedServerStream(ctx context.Context, rw http.ResponseWriter, pageSize int) (*PagedServerStream, error) {
	if pageSize == 0 {
		pageSize = 50
	}

	return &PagedServerStream{
		ctx:      ctx,
		rw:       rw,
		pageSize: pageSize,
	}, nil
}

type PagedServerStream struct {
	ctx      context.Context
	rw       http.ResponseWriter
	pageSize int

	mtx          sync.Mutex
	err          error
	headerSent   bool
	messagesSent int
	writeBuffer  bytes.Buffer
	trailer      metadata.MD
}

// Context returns the context for this stream.
func (s *PagedServerStream) Context() context.Context {
	return s.ctx
}

// SendMsg blocks until it sends m, the stream is done or the stream
// breaks.
// On error, it aborts the stream and returns an RPC status on client
// side. On server side, it simply returns the error to the caller.
// SendMsg is called by generated code.
func (s *PagedServerStream) SendMsg(m interface{}) error {
	var (
		messagesSent int
		marshaler    jsonpb.Marshaler
	)

	s.mtx.Lock()
	defer s.mtx.Unlock()

	if s.err != nil {
		return s.err
	}

	messagesSent = s.messagesSent
	s.messagesSent++

	if messagesSent >= s.pageSize {
		return io.EOF
	}
	if messagesSent == 0 {
		s.writeBuffer.WriteString(`{"items":[`)
	} else {
		s.writeBuffer.WriteString(`,`)
	}

	err := marshaler.Marshal(&s.writeBuffer, m.(proto.Message))
	if err != nil {
		s.err = err
		return err
	}

	return nil
}

// RecvMsg blocks until it receives a message or the stream is
// done. On client side, it returns io.EOF when the stream is done. On
// any other error, it aborts the streama nd returns an RPC status. On
// server side, it simply returns the error to the caller.
func (s *PagedServerStream) RecvMsg(m interface{}) error {
	return io.EOF
}

// SendHeader sends the header metadata. It should not be called
// after SendProto. It fails if called multiple times or if
// called after SendProto.
func (s *PagedServerStream) SendHeader(md metadata.MD) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if s.headerSent {
		return errors.New("unable to call SendHeader multiple times")
	}
	if s.messagesSent > 0 {
		return errors.New("unable to call SendHeader after SendMsg")
	}

	// record call to headerSent
	s.headerSent = true

	// Apply headers
	for k, v := range md {
		for _, vv := range v {
			s.rw.Header().Add(k, vv)
		}
	}

	return nil
}

// SetTrailer sets the trailer metadata which will be sent with the
// RPC status.
func (s *PagedServerStream) SetTrailer(md metadata.MD) {
	if md.Len() == 0 {
		return
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()

	s.trailer = md
}

func (s *PagedServerStream) Close() {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if s.err != nil {
		respondWithGRPCError(s.rw, s.err)
		return
	}

	if s.messagesSent == 0 {
		s.writeBuffer.WriteString(`{"items":[`)
	}

	s.writeBuffer.WriteString(`]`)

	if len(s.trailer) > 0 {
		s.writeBuffer.WriteString(`,"trailer":`)
		err := json.NewEncoder(&s.writeBuffer).Encode(s.trailer)
		if err != nil {
			respondWithGRPCError(s.rw, err)
			return
		}
	}

	s.writeBuffer.WriteString("}\n")

	s.rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	s.rw.WriteHeader(200)
	s.rw.Write(s.writeBuffer.Bytes())
}

func respondWithGRPCError(w http.ResponseWriter, err error) {
	const fallback = `{"error": "failed to marshal error message"}`

	var (
		code   = grpc.Code(err)
		desc   = grpc.ErrorDesc(err)
		status = httpStatusFromCode(code)
		msg    struct {
			Error string `json:"error"`
		}
	)

	msg.Error = desc

	data, err := json.Marshal(&msg)
	if err != nil {
		log.Logger.Errorf("error: %s", err)
		status = http.StatusInternalServerError
		data = []byte(`{"error": "failed to marshal error message"}`)
		err = nil
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	w.Write(data)
}

func httpStatusFromCode(code codes.Code) int {
	switch code {
	case codes.OK:
		return http.StatusOK
	case codes.Canceled:
		return http.StatusRequestTimeout
	case codes.Unknown:
		return http.StatusInternalServerError
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.DeadlineExceeded:
		return http.StatusRequestTimeout
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.ResourceExhausted:
		return http.StatusForbidden
	case codes.FailedPrecondition:
		return http.StatusPreconditionFailed
	case codes.Aborted:
		return http.StatusConflict
	case codes.OutOfRange:
		return http.StatusBadRequest
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Internal:
		return http.StatusInternalServerError
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DataLoss:
		return http.StatusInternalServerError
	}

	log.Logger.Errorf("Unknown gRPC error code: %v", code)
	return http.StatusInternalServerError
}

func GetFormat(field *pb.FieldDescriptorProto) string {
	if field == nil || field.Options == nil {
		return ""
	}
	v, _ := proto.GetExtension(field.Options, E_Format)
	s, _ := v.(*string)
	if s == nil {
		return ""
	}
	return *s
}
