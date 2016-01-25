// Code generated by protoc-gen-gogo.
// source: github.com/fd/featherhead/tools/plugins/svchttp/tests/example.proto
// DO NOT EDIT!

/*
	Package tests is a generated protocol buffer package.

	It is generated from these files:
		github.com/fd/featherhead/tools/plugins/svchttp/tests/example.proto

	It has these top-level messages:
		Person
		Greeting
		ListOptions
		FetchOptions
		Item
*/
package tests

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import _ "github.com/fd/featherhead/tools/runtime/svchttp"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

import strconv "strconv"
import golang_org_x_net_context "golang.org/x/net/context"
import net_http "net/http"
import encoding_json "encoding/json"
import github_com_fd_featherhead_tools_runtime_svchttp "github.com/fd/featherhead/tools/runtime/svchttp"
import github_com_juju_errors "github.com/juju/errors"
import github_com_fd_featherhead_pkg_api_httpapi_router "github.com/fd/featherhead/pkg/api/httpapi/router"

import io "io"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

type Person struct {
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
}

func (m *Person) Reset()         { *m = Person{} }
func (m *Person) String() string { return proto.CompactTextString(m) }
func (*Person) ProtoMessage()    {}

type Greeting struct {
	Text string `protobuf:"bytes,1,opt,name=text,proto3" json:"text,omitempty"`
}

func (m *Greeting) Reset()         { *m = Greeting{} }
func (m *Greeting) String() string { return proto.CompactTextString(m) }
func (*Greeting) ProtoMessage()    {}

type ListOptions struct {
	After int64 `protobuf:"varint,1,opt,name=after,proto3" json:"after,omitempty"`
}

func (m *ListOptions) Reset()         { *m = ListOptions{} }
func (m *ListOptions) String() string { return proto.CompactTextString(m) }
func (*ListOptions) ProtoMessage()    {}

type FetchOptions struct {
	Id int64 `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
}

func (m *FetchOptions) Reset()         { *m = FetchOptions{} }
func (m *FetchOptions) String() string { return proto.CompactTextString(m) }
func (*FetchOptions) ProtoMessage()    {}

type Item struct {
	Index int64 `protobuf:"varint,1,opt,name=index,proto3" json:"index,omitempty"`
}

func (m *Item) Reset()         { *m = Item{} }
func (m *Item) String() string { return proto.CompactTextString(m) }
func (*Item) ProtoMessage()    {}

func init() {
	proto.RegisterType((*Person)(nil), "xyz.featherhead.tests.Person")
	proto.RegisterType((*Greeting)(nil), "xyz.featherhead.tests.Greeting")
	proto.RegisterType((*ListOptions)(nil), "xyz.featherhead.tests.ListOptions")
	proto.RegisterType((*FetchOptions)(nil), "xyz.featherhead.tests.FetchOptions")
	proto.RegisterType((*Item)(nil), "xyz.featherhead.tests.Item")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// Client API for TestService service

type TestServiceClient interface {
	// Get a greeting using a Person
	Greet(ctx context.Context, in *Person, opts ...grpc.CallOption) (*Greeting, error)
	// List all indexes
	List(ctx context.Context, in *ListOptions, opts ...grpc.CallOption) (TestService_ListClient, error)
	// Fetch a person
	FetchPerson(ctx context.Context, in *FetchOptions, opts ...grpc.CallOption) (*Person, error)
}

type testServiceClient struct {
	cc *grpc.ClientConn
}

func NewTestServiceClient(cc *grpc.ClientConn) TestServiceClient {
	return &testServiceClient{cc}
}

func (c *testServiceClient) Greet(ctx context.Context, in *Person, opts ...grpc.CallOption) (*Greeting, error) {
	out := new(Greeting)
	err := grpc.Invoke(ctx, "/xyz.featherhead.tests.TestService/Greet", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *testServiceClient) List(ctx context.Context, in *ListOptions, opts ...grpc.CallOption) (TestService_ListClient, error) {
	stream, err := grpc.NewClientStream(ctx, &_TestService_serviceDesc.Streams[0], c.cc, "/xyz.featherhead.tests.TestService/List", opts...)
	if err != nil {
		return nil, err
	}
	x := &testServiceListClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type TestService_ListClient interface {
	Recv() (*Item, error)
	grpc.ClientStream
}

type testServiceListClient struct {
	grpc.ClientStream
}

func (x *testServiceListClient) Recv() (*Item, error) {
	m := new(Item)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *testServiceClient) FetchPerson(ctx context.Context, in *FetchOptions, opts ...grpc.CallOption) (*Person, error) {
	out := new(Person)
	err := grpc.Invoke(ctx, "/xyz.featherhead.tests.TestService/FetchPerson", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for TestService service

type TestServiceServer interface {
	// Get a greeting using a Person
	Greet(context.Context, *Person) (*Greeting, error)
	// List all indexes
	List(*ListOptions, TestService_ListServer) error
	// Fetch a person
	FetchPerson(context.Context, *FetchOptions) (*Person, error)
}

func RegisterTestServiceServer(s *grpc.Server, srv TestServiceServer) {
	s.RegisterService(&_TestService_serviceDesc, srv)
}

func _TestService_Greet_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error) (interface{}, error) {
	in := new(Person)
	if err := dec(in); err != nil {
		return nil, err
	}
	out, err := srv.(TestServiceServer).Greet(ctx, in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func _TestService_List_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(ListOptions)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(TestServiceServer).List(m, &testServiceListServer{stream})
}

type TestService_ListServer interface {
	Send(*Item) error
	grpc.ServerStream
}

type testServiceListServer struct {
	grpc.ServerStream
}

func (x *testServiceListServer) Send(m *Item) error {
	return x.ServerStream.SendMsg(m)
}

func _TestService_FetchPerson_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error) (interface{}, error) {
	in := new(FetchOptions)
	if err := dec(in); err != nil {
		return nil, err
	}
	out, err := srv.(TestServiceServer).FetchPerson(ctx, in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

var _TestService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "xyz.featherhead.tests.TestService",
	HandlerType: (*TestServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Greet",
			Handler:    _TestService_Greet_Handler,
		},
		{
			MethodName: "FetchPerson",
			Handler:    _TestService_FetchPerson_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "List",
			Handler:       _TestService_List_Handler,
			ServerStreams: true,
		},
	},
}

func (m *Person) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *Person) MarshalTo(data []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.Name) > 0 {
		data[i] = 0xa
		i++
		i = encodeVarintExample(data, i, uint64(len(m.Name)))
		i += copy(data[i:], m.Name)
	}
	return i, nil
}

func (m *Greeting) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *Greeting) MarshalTo(data []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.Text) > 0 {
		data[i] = 0xa
		i++
		i = encodeVarintExample(data, i, uint64(len(m.Text)))
		i += copy(data[i:], m.Text)
	}
	return i, nil
}

func (m *ListOptions) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *ListOptions) MarshalTo(data []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if m.After != 0 {
		data[i] = 0x8
		i++
		i = encodeVarintExample(data, i, uint64(m.After))
	}
	return i, nil
}

func (m *FetchOptions) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *FetchOptions) MarshalTo(data []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if m.Id != 0 {
		data[i] = 0x8
		i++
		i = encodeVarintExample(data, i, uint64(m.Id))
	}
	return i, nil
}

func (m *Item) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *Item) MarshalTo(data []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if m.Index != 0 {
		data[i] = 0x8
		i++
		i = encodeVarintExample(data, i, uint64(m.Index))
	}
	return i, nil
}

func encodeFixed64Example(data []byte, offset int, v uint64) int {
	data[offset] = uint8(v)
	data[offset+1] = uint8(v >> 8)
	data[offset+2] = uint8(v >> 16)
	data[offset+3] = uint8(v >> 24)
	data[offset+4] = uint8(v >> 32)
	data[offset+5] = uint8(v >> 40)
	data[offset+6] = uint8(v >> 48)
	data[offset+7] = uint8(v >> 56)
	return offset + 8
}
func encodeFixed32Example(data []byte, offset int, v uint32) int {
	data[offset] = uint8(v)
	data[offset+1] = uint8(v >> 8)
	data[offset+2] = uint8(v >> 16)
	data[offset+3] = uint8(v >> 24)
	return offset + 4
}
func encodeVarintExample(data []byte, offset int, v uint64) int {
	for v >= 1<<7 {
		data[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	data[offset] = uint8(v)
	return offset + 1
}
func (m *Person) Size() (n int) {
	var l int
	_ = l
	l = len(m.Name)
	if l > 0 {
		n += 1 + l + sovExample(uint64(l))
	}
	return n
}

func (m *Greeting) Size() (n int) {
	var l int
	_ = l
	l = len(m.Text)
	if l > 0 {
		n += 1 + l + sovExample(uint64(l))
	}
	return n
}

func (m *ListOptions) Size() (n int) {
	var l int
	_ = l
	if m.After != 0 {
		n += 1 + sovExample(uint64(m.After))
	}
	return n
}

func (m *FetchOptions) Size() (n int) {
	var l int
	_ = l
	if m.Id != 0 {
		n += 1 + sovExample(uint64(m.Id))
	}
	return n
}

func (m *Item) Size() (n int) {
	var l int
	_ = l
	if m.Index != 0 {
		n += 1 + sovExample(uint64(m.Index))
	}
	return n
}

func sovExample(x uint64) (n int) {
	for {
		n++
		x >>= 7
		if x == 0 {
			break
		}
	}
	return n
}
func sozExample(x uint64) (n int) {
	return sovExample(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}

type testServiceHandler struct {
	ss TestServiceServer
}

func RegisterTestServiceGateway(router *github_com_fd_featherhead_pkg_api_httpapi_router.Router, ss TestServiceServer) {
	h := &testServiceHandler{ss: ss}
	router.Addf("POST", "/greet", h._http_TestService_Greet)
	router.Addf("GET", "/list", h._http_TestService_List)
	router.Addf("GET", "/person/{id}", h._http_TestService_FetchPerson)
}

func (h *testServiceHandler) _http_TestService_Greet(ctx golang_org_x_net_context.Context, rw net_http.ResponseWriter, req *net_http.Request) error {
	if req.Method != "POST" {
		return github_com_juju_errors.MethodNotAllowedf("expected POST request")
	}

	var (
		input Person
	)

	ctx = github_com_fd_featherhead_tools_runtime_svchttp.AnnotateContext(ctx, req)

	{ // from body
		err := encoding_json.NewDecoder(req.Body).Decode(&input)
		if err != nil {
			return github_com_juju_errors.Trace(err)
		}
	}

	{ // call
		output, err := h.ss.Greet(ctx, &input)
		if err != nil {
			return github_com_juju_errors.Trace(err)
		}
		github_com_fd_featherhead_tools_runtime_svchttp.RenderMessageJSON(rw, 200, output)
		return nil
	}

	return nil
}

func (h *testServiceHandler) _http_TestService_List(ctx golang_org_x_net_context.Context, rw net_http.ResponseWriter, req *net_http.Request) error {
	if req.Method != "GET" {
		return github_com_juju_errors.MethodNotAllowedf("expected GET request")
	}

	var (
		input ListOptions
	)

	ctx = github_com_fd_featherhead_tools_runtime_svchttp.AnnotateContext(ctx, req)

	// populate after=after
	{
		var msg0 = &input
		val := req.URL.Query().Get("after")
		intVal, _ := strconv.ParseInt(val, 10, 64)
		msg0.After = int64(intVal)
	}

	{ // call
		ss, err := github_com_fd_featherhead_tools_runtime_svchttp.NewPagedServerStream(ctx, rw, 0)
		if err != nil {
			return github_com_juju_errors.Trace(err)
		}
		defer ss.Close()
		stream := &testServiceListServer{ss}
		err = h.ss.List(&input, stream)
	}

	return nil
}

func (h *testServiceHandler) _http_TestService_FetchPerson(ctx golang_org_x_net_context.Context, rw net_http.ResponseWriter, req *net_http.Request) error {
	if req.Method != "GET" {
		return github_com_juju_errors.MethodNotAllowedf("expected GET request")
	}

	var (
		input  FetchOptions
		params = github_com_fd_featherhead_pkg_api_httpapi_router.P(ctx)
	)

	ctx = github_com_fd_featherhead_tools_runtime_svchttp.AnnotateContext(ctx, req)

	// populate id
	{
		var msg0 = &input
		val := params.Get("id")
		intVal, _ := strconv.ParseInt(val, 10, 64)
		msg0.Id = int64(intVal)
	}

	{ // call
		output, err := h.ss.FetchPerson(ctx, &input)
		if err != nil {
			return github_com_juju_errors.Trace(err)
		}
		github_com_fd_featherhead_tools_runtime_svchttp.RenderMessageJSON(rw, 200, output)
		return nil
	}

	return nil
}

func (m *Person) Unmarshal(data []byte) error {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowExample
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Person: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Person: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Name", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowExample
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthExample
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Name = string(data[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipExample(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthExample
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *Greeting) Unmarshal(data []byte) error {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowExample
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Greeting: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Greeting: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Text", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowExample
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthExample
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Text = string(data[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipExample(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthExample
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *ListOptions) Unmarshal(data []byte) error {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowExample
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: ListOptions: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ListOptions: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field After", wireType)
			}
			m.After = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowExample
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				m.After |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipExample(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthExample
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *FetchOptions) Unmarshal(data []byte) error {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowExample
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: FetchOptions: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: FetchOptions: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Id", wireType)
			}
			m.Id = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowExample
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				m.Id |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipExample(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthExample
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *Item) Unmarshal(data []byte) error {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowExample
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Item: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Item: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Index", wireType)
			}
			m.Index = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowExample
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				m.Index |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipExample(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthExample
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipExample(data []byte) (n int, err error) {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowExample
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowExample
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if data[iNdEx-1] < 0x80 {
					break
				}
			}
			return iNdEx, nil
		case 1:
			iNdEx += 8
			return iNdEx, nil
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowExample
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			iNdEx += length
			if length < 0 {
				return 0, ErrInvalidLengthExample
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return 0, ErrIntOverflowExample
					}
					if iNdEx >= l {
						return 0, io.ErrUnexpectedEOF
					}
					b := data[iNdEx]
					iNdEx++
					innerWire |= (uint64(b) & 0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				innerWireType := int(innerWire & 0x7)
				if innerWireType == 4 {
					break
				}
				next, err := skipExample(data[start:])
				if err != nil {
					return 0, err
				}
				iNdEx = start + next
			}
			return iNdEx, nil
		case 4:
			return iNdEx, nil
		case 5:
			iNdEx += 4
			return iNdEx, nil
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
	}
	panic("unreachable")
}

var (
	ErrInvalidLengthExample = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowExample   = fmt.Errorf("proto: integer overflow")
)
