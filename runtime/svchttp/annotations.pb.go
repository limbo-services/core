// Code generated by protoc-gen-gogo.
// source: github.com/fd/featherhead/tools/runtime/svchttp/annotations.proto
// DO NOT EDIT!

/*
Package svchttp is a generated protocol buffer package.

It is generated from these files:
	github.com/fd/featherhead/tools/runtime/svchttp/annotations.proto
	github.com/fd/featherhead/tools/runtime/svchttp/http_rule.proto

It has these top-level messages:
	HttpRule
*/
package svchttp

import proto "github.com/gogo/protobuf/proto"
import fmt "fmt"
import math "math"
import google_protobuf "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

var E_Http = &proto.ExtensionDesc{
	ExtendedType:  (*google_protobuf.MethodOptions)(nil),
	ExtensionType: (*HttpRule)(nil),
	Field:         58702,
	Name:          "xyz.featherhead.api.http",
	Tag:           "bytes,58702,opt,name=http",
}

func init() {
	proto.RegisterExtension(E_Http)
}
