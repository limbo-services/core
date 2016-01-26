// Code generated by protoc-gen-gogo.
// source: github.com/fd/featherhead/tools/runtime/svcauth/authz.proto
// DO NOT EDIT!

package svcauth

import proto "github.com/gogo/protobuf/proto"
import fmt "fmt"
import math "math"
import _ "github.com/gogo/protobuf/gogoproto"

import strings "strings"
import github_com_gogo_protobuf_proto "github.com/gogo/protobuf/proto"
import sort "sort"
import strconv "strconv"
import reflect "reflect"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

type AuthzRule struct {
	// Path to caller identifier.
	Caller string `protobuf:"bytes,1,opt,name=caller,proto3" json:"caller,omitempty"`
	// Path to context identifier.
	Context string `protobuf:"bytes,2,opt,name=context,proto3" json:"context,omitempty"`
	// Scopes that give permission to call the RPC
	Scopes []string `protobuf:"bytes,3,rep,name=scope" json:"scope,omitempty"`
}

func (m *AuthzRule) Reset()         { *m = AuthzRule{} }
func (m *AuthzRule) String() string { return proto.CompactTextString(m) }
func (*AuthzRule) ProtoMessage()    {}

func init() {
	proto.RegisterType((*AuthzRule)(nil), "xyz.featherhead.api.AuthzRule")
}
func (this *AuthzRule) GoString() string {
	if this == nil {
		return "nil"
	}
	s := make([]string, 0, 7)
	s = append(s, "&svcauth.AuthzRule{")
	s = append(s, "Caller: "+fmt.Sprintf("%#v", this.Caller)+",\n")
	s = append(s, "Context: "+fmt.Sprintf("%#v", this.Context)+",\n")
	s = append(s, "Scopes: "+fmt.Sprintf("%#v", this.Scopes)+",\n")
	s = append(s, "}")
	return strings.Join(s, "")
}
func valueToGoStringAuthz(v interface{}, typ string) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("func(v %v) *%v { return &v } ( %#v )", typ, typ, pv)
}
func extensionToGoStringAuthz(e map[int32]github_com_gogo_protobuf_proto.Extension) string {
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