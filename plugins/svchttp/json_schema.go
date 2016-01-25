package svchttp

import (
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

	case pb.FieldDescriptorProto_TYPE_MESSAGE:
		messageType := gen.ObjectNamed(field.GetTypeName()).(*generator.Descriptor)
		return messageToSchema(gen, messageType)

	}

	panic("unsupported " + field.GetType().String())
}
