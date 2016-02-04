package generator

import (
	"github.com/limbo-services/protobuf/gogoproto"
	"github.com/limbo-services/protobuf/proto"
	pb "github.com/limbo-services/protobuf/protoc-gen-gogo/descriptor"
)

var (
	step1 = SetBoolFieldOption(gogoproto.E_Nullable, false)
	step2 = SetStringFieldOption(gogoproto.E_Casttype, "time.Time")
)

func MapTimestampToTime(file *pb.FileDescriptorProto) {
	for _, msg := range file.MessageType {
		mapTimestamp(msg)
	}
}

func mapTimestamp(msg *pb.DescriptorProto) {
	for _, field := range msg.Field {
		if field.GetTypeName() == ".google.protobuf.Timestamp" {
			step1(field)
			step2(field)
		}
	}
	for _, msg := range msg.NestedType {
		mapTimestamp(msg)
	}
}

func FieldHasBoolExtension(field *pb.FieldDescriptorProto, extension *proto.ExtensionDesc) bool {
	if field.Options == nil {
		return false
	}
	value, err := proto.GetExtension(field.Options, extension)
	if err != nil {
		return false
	}
	if value == nil {
		return false
	}
	if value.(*bool) == nil {
		return false
	}
	return true
}

func SetBoolFieldOption(extension *proto.ExtensionDesc, value bool) func(field *pb.FieldDescriptorProto) {
	return func(field *pb.FieldDescriptorProto) {
		if FieldHasBoolExtension(field, extension) {
			return
		}
		if field.Options == nil {
			field.Options = &pb.FieldOptions{}
		}
		if err := proto.SetExtension(field.Options, extension, &value); err != nil {
			panic(err)
		}
	}
}

func FieldHasStringExtension(field *pb.FieldDescriptorProto, extension *proto.ExtensionDesc) bool {
	if field.Options == nil {
		return false
	}
	value, err := proto.GetExtension(field.Options, extension)
	if err != nil {
		return false
	}
	if value == nil {
		return false
	}
	if value.(*string) == nil {
		return false
	}
	return true
}

func SetStringFieldOption(extension *proto.ExtensionDesc, value string) func(field *pb.FieldDescriptorProto) {
	return func(field *pb.FieldDescriptorProto) {
		if FieldHasStringExtension(field, extension) {
			return
		}
		if field.Options == nil {
			field.Options = &pb.FieldOptions{}
		}
		if err := proto.SetExtension(field.Options, extension, &value); err != nil {
			panic(err)
		}
	}
}
