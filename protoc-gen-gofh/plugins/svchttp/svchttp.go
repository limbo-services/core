package svchttp

import (
	"fmt"
	"os"
	"strings"

	"github.com/fd/featherhead/proto"
	"github.com/gogo/protobuf/proto"
	pb "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
)

// Paths for packages used by code generated in this file,
// relative to the import_prefix of the generator.Generator.
const (
	contextPkgPath   = "golang.org/x/net/context"
	grpcPkgPath      = "google.golang.org/grpc"
	grpcCodesPkgPath = "google.golang.org/grpc/codes"
)

func init() {
	generator.RegisterPlugin(new(svchttp))
}

// grpc is an implementation of the Go protocol buffer compiler's
// plugin architecture.  It generates bindings for gRPC support.
type svchttp struct {
	gen *generator.Generator
}

// Name returns the name of this plugin, "grpc".
func (g *svchttp) Name() string {
	return "svchttp"
}

// reservedClientName records whether a client name is reserved on the client side.
var reservedClientName = map[string]bool{
// TODO: do we need any in gRPC?
}

// Init initializes the plugin.
func (g *svchttp) Init(gen *generator.Generator) {
	g.gen = gen
}

// Given a type name defined in a .proto, return its object.
// Also record that we're using it, to guarantee the associated import.
func (g *svchttp) objectNamed(name string) generator.Object {
	g.gen.RecordTypeUse(name)
	return g.gen.ObjectNamed(name)
}

// Given a type name defined in a .proto, return its name as we will print it.
func (g *svchttp) typeName(str string) string {
	return g.gen.TypeName(g.objectNamed(str))
}

// P forwards to g.gen.P.
func (g *svchttp) P(args ...interface{}) { g.gen.P(args...) }

// Generate generates code for the services in the given file.
func (g *svchttp) Generate(file *generator.FileDescriptor) {
	if len(file.FileDescriptorProto.Service) == 0 {
		return
	}
	g.P("// Reference imports to suppress errors if they are not otherwise used.")
	g.P("var _ context_svchttp.Context")
	g.P("var _ grpc_svchttp.Codec")
	g.P("var _ codes_svchttp.Code")
	g.P()
	for i, service := range file.FileDescriptorProto.Service {
		g.generateService(file, service, i)
	}
}

// GenerateImports generates the import declaration for this file.
func (g *svchttp) GenerateImports(file *generator.FileDescriptor) {
	if len(file.FileDescriptorProto.Service) == 0 {
		return
	}
	g.gen.PrintImport("context_svchttp", contextPkgPath)
	g.gen.PrintImport("grpc_svchttp", grpcPkgPath)
	g.gen.PrintImport("codes_svchttp", grpcCodesPkgPath)
}

func unexport(s string) string { return strings.ToLower(s[:1]) + s[1:] }

// generateService generates all the code for the named service.
func (g *svchttp) generateService(file *generator.FileDescriptor, service *pb.ServiceDescriptorProto, index int) {

	// origServName := service.GetName()
	// servName := generator.CamelCase(origServName)

	// Server interface.
	// innerServerType := servName + "Server"
	// serverType := unexport(servName) + "ServerGateway"

	// Server handler implementations.
	for _, method := range service.Method {
		v, _ := proto.GetExtension(method.Options, fhannotations.E_Http)
		info, _ := v.(*fhannotations.HttpRule)
		if info == nil {
			continue
		}

		inputTypeName := method.GetInputType()
		inputType, _ := g.gen.ObjectNamed(inputTypeName).(*generator.Descriptor)

		outputTypeName := method.GetOutputType()
		outputType, _ := g.gen.ObjectNamed(outputTypeName).(*generator.Descriptor)

		//g.gen.GetFieldName(message *generator.Descriptor, field *descriptor.FieldDescriptorProto)

		fmt.Fprintf(os.Stderr, "info: %v\n", info)
		fmt.Fprintf(os.Stderr, "  input:  %v\n", inputType)
		fmt.Fprintf(os.Stderr, "  output: %v\n", outputType)
	}

}

// generateServerSignature returns the server-side signature for a method.
func (g *svchttp) generateServerSignature(servName string, method *pb.MethodDescriptorProto) string {
	origMethName := method.GetName()
	methName := generator.CamelCase(origMethName)
	if reservedClientName[methName] {
		methName += "_"
	}

	var reqArgs []string
	ret := "(err error)"
	if !method.GetServerStreaming() && !method.GetClientStreaming() {
		reqArgs = append(reqArgs, "ctx context_svchttp.Context")
		ret = "(out *" + g.typeName(method.GetOutputType()) + ", err error)"
	}
	if !method.GetClientStreaming() {
		reqArgs = append(reqArgs, "input *"+g.typeName(method.GetInputType()))
	}
	if method.GetServerStreaming() || method.GetClientStreaming() {
		reqArgs = append(reqArgs, "stream "+servName+"_"+generator.CamelCase(origMethName)+"Server")
	}

	return methName + "(" + strings.Join(reqArgs, ", ") + ") " + ret
}

// generateServerSignature returns the server-side signature for a method.
func (g *svchttp) generateServerCall(servName string, method *pb.MethodDescriptorProto) string {
	origMethName := method.GetName()
	methName := generator.CamelCase(origMethName)
	if reservedClientName[methName] {
		methName += "_"
	}

	var reqArgs []string
	if !method.GetServerStreaming() && !method.GetClientStreaming() {
		reqArgs = append(reqArgs, "ctx")
	}
	if !method.GetClientStreaming() {
		reqArgs = append(reqArgs, "input")
	}
	if method.GetServerStreaming() || method.GetClientStreaming() {
		reqArgs = append(reqArgs, "stream")
	}

	return methName + "(" + strings.Join(reqArgs, ", ") + ") "
}
