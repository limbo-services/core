package svchttp

import (
	"fmt"
	"strings"

	pb "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
)

func messageToSchema(gen *generator.Generator, desc *generator.Descriptor) interface{} {

	properties := make(map[string]interface{})
	for _, field := range desc.GetField() {
		// required/optional/repeated
		properties[field.GetName()] = fieldToSchema(gen, field)
	}

	schema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}

	return schema
}

func fieldToSchema(gen *generator.Generator, field *pb.FieldDescriptorProto) interface{} {
	switch field.GetType() {

	case pb.FieldDescriptorProto_TYPE_BOOL:
		return map[string]interface{}{
			"type": "boolean",
		}

	case pb.FieldDescriptorProto_TYPE_DOUBLE,
		pb.FieldDescriptorProto_TYPE_FLOAT:
		return map[string]interface{}{
			"type": "number",
		}

	case pb.FieldDescriptorProto_TYPE_INT32,
		pb.FieldDescriptorProto_TYPE_INT64,
		pb.FieldDescriptorProto_TYPE_FIXED32,
		pb.FieldDescriptorProto_TYPE_FIXED64,
		pb.FieldDescriptorProto_TYPE_SFIXED32,
		pb.FieldDescriptorProto_TYPE_SFIXED64,
		pb.FieldDescriptorProto_TYPE_SINT32,
		pb.FieldDescriptorProto_TYPE_SINT64,
		pb.FieldDescriptorProto_TYPE_UINT32,
		pb.FieldDescriptorProto_TYPE_UINT64:
		return map[string]interface{}{
			"type": "integer",
		}

	case pb.FieldDescriptorProto_TYPE_STRING:
		return map[string]interface{}{
			"type": "string",
		}

	case pb.FieldDescriptorProto_TYPE_BYTES:
		return map[string]interface{}{
			"type":   "string",
			"format": "base64",
		}

	case pb.FieldDescriptorProto_TYPE_ENUM:
		switch messageType := gen.ObjectNamed(field.GetTypeName()).(type) {

		case *generator.EnumDescriptor:
			values := make([]interface{}, 2*len(messageType.Value))
			for _, x := range messageType.Value {
				values = append(values, x.GetNumber())
				values = append(values, x.GetName())
			}
			return map[string]interface{}{
				"enum": values,
			}

		default:
			panic(fmt.Sprintf("unsuported ENUM %T", messageType))

		}

	case pb.FieldDescriptorProto_TYPE_MESSAGE:
		return map[string]interface{}{
			"$ref": "#/definitions/" + strings.TrimPrefix(field.GetTypeName(), "."),
		}

	}

	panic("unsupported " + field.GetType().String())
}