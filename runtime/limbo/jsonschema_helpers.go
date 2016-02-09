package limbo

import (
	"bytes"
	"encoding/json"

	"github.com/limbo-services/protobuf/proto"
	pb "github.com/limbo-services/protobuf/protoc-gen-gogo/descriptor"
)

var definitions = map[string]SchemaDefinition{}

type SchemaDefinition struct {
	Name         string
	Definition   []byte
	Dependencies []string
}

func RegisterSchemaDefinitions(defs []SchemaDefinition) struct{} {
	var buf bytes.Buffer
	for _, def := range defs {
		buf.Reset()
		if err := json.Compact(&buf, def.Definition); err == nil {
			def.Definition = append(def.Definition[:0], buf.Bytes()...)
		}
		definitions[def.Name] = def
	}
	return struct{}{}
}

func IsRequiredProperty(field *pb.FieldDescriptorProto) bool {
	if field.Options != nil && proto.HasExtension(field.Options, E_Required) {
		return proto.GetBoolExtension(field.Options, E_Required, false)
	}
	if field.IsRequired() {
		return true
	}
	return false
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

func GetPattern(field *pb.FieldDescriptorProto) (string, bool) {
	if field == nil || field.Options == nil {
		return "", false
	}
	v, _ := proto.GetExtension(field.Options, E_Pattern)
	s, _ := v.(*string)
	if s == nil {
		return "", false
	}
	return *s, true
}
