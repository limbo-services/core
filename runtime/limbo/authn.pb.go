// Code generated by protoc-gen-gogo.
// source: limbo.services/core/runtime/limbo/authn.proto
// DO NOT EDIT!

package limbo

import proto "limbo.services/protobuf/proto"
import fmt "fmt"
import math "math"
import _ "limbo.services/protobuf/gogoproto"

import strings "strings"
import limbo_services_protobuf_proto "limbo.services/protobuf/proto"
import sort "sort"
import strconv "strconv"
import reflect "reflect"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

type AuthnRule struct {
	// Path to caller field
	Caller string `protobuf:"bytes,1,opt,name=caller,proto3" json:"caller,omitempty"`
	// Strategies used to authenticate.
	Strategies []string `protobuf:"bytes,2,rep,name=strategy" json:"strategy,omitempty"`
}

func (m *AuthnRule) Reset()                    { *m = AuthnRule{} }
func (m *AuthnRule) String() string            { return proto.CompactTextString(m) }
func (*AuthnRule) ProtoMessage()               {}
func (*AuthnRule) Descriptor() ([]byte, []int) { return fileDescriptorAuthn, []int{0} }

func (this *AuthnRule) GoString() string {
	if this == nil {
		return "nil"
	}
	s := make([]string, 0, 6)
	s = append(s, "&limbo.AuthnRule{")
	s = append(s, "Caller: "+fmt.Sprintf("%#v", this.Caller)+",\n")
	s = append(s, "Strategies: "+fmt.Sprintf("%#v", this.Strategies)+",\n")
	s = append(s, "}")
	return strings.Join(s, "")
}
func valueToGoStringAuthn(v interface{}, typ string) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("func(v %v) *%v { return &v } ( %#v )", typ, typ, pv)
}
func extensionToGoStringAuthn(e map[int32]limbo_services_protobuf_proto.Extension) string {
	if e == nil {
		return "nil"
	}
	s := "map[int32]proto.Extension{"
	keys := make([]int, 0, len(e))
	for k := range e {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	ss := []string{}
	for _, k := range keys {
		ss = append(ss, strconv.Itoa(k)+": "+e[int32(k)].GoString())
	}
	s += strings.Join(ss, ",") + "}"
	return s
}
func init() {
	proto.RegisterType((*AuthnRule)(nil), "limbo.AuthnRule")
}

var fileDescriptorAuthn = []byte{
	// 171 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0xe2, 0xd2, 0xcd, 0xc9, 0xcc, 0x4d,
	0xca, 0xd7, 0x2b, 0x4e, 0x2d, 0x2a, 0xcb, 0x4c, 0x4e, 0x2d, 0xd6, 0x4f, 0xce, 0x2f, 0x4a, 0xd5,
	0x2f, 0x2a, 0xcd, 0x2b, 0xc9, 0xcc, 0x4d, 0xd5, 0x07, 0xcb, 0xe9, 0x27, 0x96, 0x96, 0x64, 0xe4,
	0xe9, 0x15, 0x14, 0xe5, 0x97, 0xe4, 0x0b, 0xb1, 0x82, 0x85, 0xa4, 0x74, 0xd0, 0x74, 0x81, 0x25,
	0x93, 0x4a, 0xd3, 0xf4, 0xd3, 0xf3, 0xd3, 0xf3, 0xc1, 0x1c, 0x30, 0x0b, 0xa2, 0x49, 0x29, 0x94,
	0x8b, 0xd3, 0x11, 0x64, 0x46, 0x50, 0x69, 0x4e, 0xaa, 0x90, 0x18, 0x17, 0x5b, 0x72, 0x62, 0x4e,
	0x4e, 0x6a, 0x91, 0x04, 0xa3, 0x02, 0xa3, 0x06, 0x67, 0x10, 0x94, 0x27, 0xa4, 0xc5, 0xc5, 0x51,
	0x5c, 0x52, 0x94, 0x58, 0x92, 0x9a, 0x5e, 0x29, 0xc1, 0xa4, 0xc0, 0xac, 0xc1, 0xe9, 0xc4, 0xf7,
	0xe8, 0x9e, 0x3c, 0x57, 0x30, 0x44, 0x2c, 0x33, 0xb5, 0x38, 0x08, 0x2e, 0x6f, 0xc5, 0xb2, 0x61,
	0x81, 0x3c, 0xa3, 0x13, 0x7b, 0x14, 0xc4, 0x35, 0x49, 0x6c, 0x60, 0x6b, 0x8c, 0x01, 0x01, 0x00,
	0x00, 0xff, 0xff, 0x1d, 0x73, 0x42, 0xd2, 0xcc, 0x00, 0x00, 0x00,
}
