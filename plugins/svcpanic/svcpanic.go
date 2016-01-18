package svcpanic

import (
	"strings"

	pb "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
)

// Paths for packages used by code generated in this file,
// relative to the import_prefix of the generator.Generator.
const (
	contextPkgPath = "golang.org/x/net/context"
	grpcPkgPath    = "google.golang.org/grpc"
	runtimePkgPath = "github.com/fd/featherhead/tools/runtime/svcpanic"
)

func init() {
	generator.RegisterPlugin(new(svcpanic))
}

// grpc is an implementation of the Go protocol buffer compiler's
// plugin architecture.  It generates bindings for gRPC support.
type svcpanic struct {
	gen *generator.Generator

	imports    generator.PluginImports
	contextPkg generator.Single
	grpcPkg    generator.Single
	runtimePkg generator.Single
}

// Name returns the name of this plugin, "grpc".
func (g *svcpanic) Name() string {
	return "svcpanic"
}

// reservedClientName records whether a client name is reserved on the client side.
var reservedClientName = map[string]bool{
// TODO: do we need any in gRPC?
}

// Init initializes the plugin.
func (g *svcpanic) Init(gen *generator.Generator) {
	g.gen = gen
}

// Given a type name defined in a .proto, return its object.
// Also record that we're using it, to guarantee the associated import.
func (g *svcpanic) objectNamed(name string) generator.Object {
	g.gen.RecordTypeUse(name)
	return g.gen.ObjectNamed(name)
}

// Given a type name defined in a .proto, return its name as we will print it.
func (g *svcpanic) typeName(str string) string {
	return g.gen.TypeName(g.objectNamed(str))
}

// P forwards to g.gen.P.
func (g *svcpanic) P(args ...interface{}) { g.gen.P(args...) }

// Generate generates code for the services in the given file.
func (g *svcpanic) Generate(file *generator.FileDescriptor) {
	if len(file.FileDescriptorProto.Service) == 0 {
		return
	}

	imp := generator.NewPluginImports(g.gen)
	g.imports = imp
	g.contextPkg = imp.NewImport(contextPkgPath)
	g.grpcPkg = imp.NewImport(grpcPkgPath)
	g.runtimePkg = imp.NewImport(runtimePkgPath)

	for i, service := range file.FileDescriptorProto.Service {
		g.generateService(file, service, i)
	}
}

// GenerateImports generates the import declaration for this file.
func (g *svcpanic) GenerateImports(file *generator.FileDescriptor) {
	if len(file.FileDescriptorProto.Service) == 0 {
		return
	}

	g.imports.GenerateImports(file)
}

func unexport(s string) string { return strings.ToLower(s[:1]) + s[1:] }

// generateService generates all the code for the named service.
func (g *svcpanic) generateService(file *generator.FileDescriptor, service *pb.ServiceDescriptorProto, index int) {

	origServName := service.GetName()
	servName := generator.CamelCase(origServName)

	// Server interface.
	innerServerType := servName + "Server"
	serverType := unexport(servName) + "ServerPanicGuard"
	g.P("type ", serverType, " struct {")
	g.P(`handler `, g.runtimePkg.Use(), `.ErrorHandler`)
	g.P(`inner `, innerServerType)
	g.P("}")
	g.P()

	g.P("func New", servName, "ServerPanicGuard(inner ", innerServerType, `, handler `, g.runtimePkg.Use(), `.ErrorHandler) `, innerServerType, " {")
	g.P("return &", serverType, "{ inner : inner, handler: handler}")
	g.P("}")
	g.P()

	// Server handler implementations.
	for _, method := range service.Method {
		g.P(`func (s *`, serverType, `) `, g.generateServerSignature(servName, method), ` {`)
		g.P(`defer `, g.runtimePkg.Use(), `.RecoverPanic(&err, s.handler)`)
		g.P(`return s.inner.`, g.generateServerCall(servName, method))
		g.P(`}`)
		g.P()
	}

}

// generateServerSignature returns the server-side signature for a method.
func (g *svcpanic) generateServerSignature(servName string, method *pb.MethodDescriptorProto) string {
	origMethName := method.GetName()
	methName := generator.CamelCase(origMethName)
	if reservedClientName[methName] {
		methName += "_"
	}

	var reqArgs []string
	ret := "(err error)"
	if !method.GetServerStreaming() && !method.GetClientStreaming() {
		reqArgs = append(reqArgs, `ctx `+g.contextPkg.Use()+`.Context`)
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
func (g *svcpanic) generateServerCall(servName string, method *pb.MethodDescriptorProto) string {
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
