package svcauth

import (
	"fmt"
	"path"
	"strings"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	pb "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
	plugin "github.com/gogo/protobuf/protoc-gen-gogo/plugin"

	grpcutil "github.com/fd/featherhead/pkg/api/grpcutil"
)

// Paths for packages used by code generated in this file,
// relative to the import_prefix of the generator.Generator.
const (
	contextPkgPath   = "golang.org/x/net/context"
	grpcPkgPath      = "google.golang.org/grpc"
	grpcCodesPkgPath = "google.golang.org/grpc/codes"
	grpcutilPkgPath  = "github.com/fd/featherhead/pkg/api/grpcutil"
)

func init() {
	generator.RegisterPlugin(new(svcauth))
}

// grpc is an implementation of the Go protocol buffer compiler's
// plugin architecture.  It generates bindings for gRPC support.
type svcauth struct {
	gen *generator.Generator
}

// Name returns the name of this plugin, "grpc".
func (g *svcauth) Name() string {
	return "svcauth"
}

// reservedClientName records whether a client name is reserved on the client side.
var reservedClientName = map[string]bool{
// TODO: do we need any in gRPC?
}

// Init initializes the plugin.
func (g *svcauth) Init(gen *generator.Generator) {
	g.gen = gen
}

// Given a type name defined in a .proto, return its object.
// Also record that we're using it, to guarantee the associated import.
func (g *svcauth) objectNamed(name string) generator.Object {
	g.gen.RecordTypeUse(name)
	return g.gen.ObjectNamed(name)
}

// Given a type name defined in a .proto, return its name as we will print it.
func (g *svcauth) typeName(str string) string {
	return g.gen.TypeName(g.objectNamed(str))
}

// P forwards to g.gen.P.
func (g *svcauth) P(args ...interface{}) { g.gen.P(args...) }

// Generate generates code for the services in the given file.
func (g *svcauth) Generate(file *generator.FileDescriptor) {
	if len(file.FileDescriptorProto.Service) == 0 {
		return
	}
	g.P("// Reference imports to suppress errors if they are not otherwise used.")
	g.P("var _ context_svcauth.Context")
	g.P("var _ grpc_svcauth.Codec")
	g.P("var _ codes_svcauth.Code")
	g.P()
	for i, service := range file.FileDescriptorProto.Service {
		g.generateService(file, service, i)
	}
}

// GenerateImports generates the import declaration for this file.
func (g *svcauth) GenerateImports(file *generator.FileDescriptor) {
	if len(file.FileDescriptorProto.Service) == 0 {
		return
	}
	g.gen.PrintImport("context_svcauth", contextPkgPath)
	g.gen.PrintImport("grpc_svcauth", grpcPkgPath)
	g.gen.PrintImport("codes_svcauth", grpcCodesPkgPath)
	g.gen.PrintImport("grpcutil_svcauth", grpcutilPkgPath)
}

func unexport(s string) string { return strings.ToLower(s[:1]) + s[1:] }

// generateService generates all the code for the named service.
func (g *svcauth) generateService(file *generator.FileDescriptor, service *pb.ServiceDescriptorProto, index int) {
	methods := g.findMethods(file, service)
	if len(methods) == 0 {
		return
	}

	origServName := service.GetName()
	servName := generator.CamelCase(origServName)

	// Server interface.
	innerServerType := servName + "Server"
	serverType := unexport(servName) + "ServerAuthGuard"
	g.P("type ", serverType, " struct {")
	g.P("authenticator grpcutil_svcauth.Authenticator")
	g.P("inner ", innerServerType)
	g.P("}")
	g.P()

	g.P("var _ ", innerServerType, " = (*", serverType, ")(nil)")
	g.P()

	g.P("func New", servName, "ServerAuthGuard(inner ", innerServerType, ", auth grpcutil_svcauth.Authenticator) ", innerServerType, " {")
	g.P("return &", serverType, "{inner: inner, authenticator: auth}")
	g.P("}")
	g.P()

	// Server handler implementations.
	for _, authMethod := range methods {
		method, authnInfo, authzInfo := authMethod.method, authMethod.Authn, authMethod.Authz

		authnRuleName := g.generateAuthnRuleName(servName, method)
		authzRuleName := g.generateAuthzRuleName(servName, method)
		g.P("var (")
		g.P(authnRuleName, " *grpcutil_svcauth.AuthnRule = ", replaceFhAnnotationNames(authnInfo.GoString()))
		g.P(authzRuleName, " *grpcutil_svcauth.AuthzRule = ", replaceFhAnnotationNames(authzInfo.GoString()))
		g.P(")")
		g.P("")

		g.P("func (s *", serverType, ") ", g.generateServerSignature(servName, method), " {")
		g.P("var (")
		g.P("info grpcutil_svcauth.AuthInfo")
		if method.GetServerStreaming() || method.GetClientStreaming() {
			g.P("ctx = stream.Context()")

			streamName := servName + generator.CamelCase(method.GetName()) + "Server"
			streamPtrName := unexport(streamName)
			g.P(`streamPtr = stream.(*`, streamPtrName, `)`)
		}
		g.P(")")

		g.P("if err := s.authenticator.Authn(ctx, ", authnRuleName, ", &info); err != nil {")
		if !method.GetServerStreaming() && !method.GetClientStreaming() {
			g.P("return nil, err")
		} else {
			g.P("return err")
		}
		g.P("}")

		if authzInfo != nil {
			g.P("if err := s.authenticator.Authz(ctx, ", authzRuleName, ", &info); err != nil {")
			if !method.GetServerStreaming() && !method.GetClientStreaming() {
				g.P("return nil, err")
			} else {
				g.P("return err")
			}
			g.P("}")
			g.P("")
		}

		g.P(`ctx = grpcutil_svcauth.ContextWithAuthInfo(ctx, &info)`)

		if method.GetServerStreaming() || method.GetClientStreaming() {
			g.P(`streamPtr.ServerStream = grpcutil_svcauth.WrapServerStreamContext(streamPtr.ServerStream, ctx)`)
		}

		g.P("return s.inner.", g.generateServerCall(servName, method))
		g.P("}")
		g.P()
	}

	{
		var desc = grpcutil.AuthDescriptionSet{}

		for _, m := range methods {
			desc.Methods = append(desc.Methods, &grpcutil.AuthDescription{
				Method: m.Name,
				Authn:  m.Authn,
				Authz:  m.Authz,
			})
		}

		m := jsonpb.Marshaler{}
		data, err := m.MarshalToString(&desc)
		if err != nil {
			g.gen.Error(err)
		}

		g.gen.Response.File = append(g.gen.Response.File, &plugin.CodeGeneratorResponse_File{
			Name:    proto.String(authSpecFileName(*file.Name)),
			Content: proto.String(data),
		})
	}

}

// generateServerSignature returns the server-side signature for a method.
func (g *svcauth) generateServerSignature(servName string, method *pb.MethodDescriptorProto) string {
	origMethName := method.GetName()
	methName := generator.CamelCase(origMethName)
	if reservedClientName[methName] {
		methName += "_"
	}

	var reqArgs []string
	ret := "error"
	if !method.GetServerStreaming() && !method.GetClientStreaming() {
		reqArgs = append(reqArgs, "ctx context_svcauth.Context")
		ret = "(*" + g.typeName(method.GetOutputType()) + ", error)"
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
func (g *svcauth) generateServerCall(servName string, method *pb.MethodDescriptorProto) string {
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

func (g *svcauth) generateAuthnRuleName(servName string, method *pb.MethodDescriptorProto) string {
	origMethName := method.GetName()
	methName := generator.CamelCase(origMethName)
	if reservedClientName[methName] {
		methName += "_"
	}

	return "authn_" + servName + "_" + methName
}

func (g *svcauth) generateAuthzRuleName(servName string, method *pb.MethodDescriptorProto) string {
	origMethName := method.GetName()
	methName := generator.CamelCase(origMethName)
	if reservedClientName[methName] {
		methName += "_"
	}

	return "authz_" + servName + "_" + methName
}

func replaceFhAnnotationNames(s string) string {
	return strings.Replace(s, "grpcutil.", "grpcutil_svcauth.", -1)
}

type authMethod struct {
	file    *generator.FileDescriptor
	service *pb.ServiceDescriptorProto
	method  *pb.MethodDescriptorProto
	Name    string              `json:"method"`
	Authn   *grpcutil.AuthnRule `json:"authn"`
	Authz   *grpcutil.AuthzRule `json:"authz" `
}

func (g *svcauth) findMethods(file *generator.FileDescriptor, service *pb.ServiceDescriptorProto) []*authMethod {
	methods := make([]*authMethod, 0, len(service.Method))

	for _, method := range service.Method {
		var (
			authnInfo *grpcutil.AuthnRule
			authzInfo *grpcutil.AuthzRule
		)

		{ // authn
			v, _ := proto.GetExtension(method.Options, grpcutil.E_Authn)
			authnInfo, _ = v.(*grpcutil.AuthnRule)
			if authnInfo == nil {
				authnInfo = &grpcutil.AuthnRule{}
				authnInfo.Gateway = grpcutil.AuthnRule_DENY
			}
			authnInfo.SetDefaults()
		}

		{ // authz
			v, _ := proto.GetExtension(method.Options, grpcutil.E_Authz)
			authzInfo, _ = v.(*grpcutil.AuthzRule)
			if authzInfo != nil {
				authzInfo.SetDefaults()
			}
		}

		methods = append(methods, &authMethod{
			file:    file,
			service: service,
			method:  method,
			Authn:   authnInfo,
			Authz:   authzInfo,
			Name:    fmt.Sprintf("/%s.%s/%s", file.GetPackage(), service.GetName(), method.GetName()),
		})
	}

	return methods
}

func authSpecFileName(name string) string {
	ext := path.Ext(name)
	if ext == ".proto" || ext == ".protodevel" {
		name = name[0 : len(name)-len(ext)]
	}
	return name + ".auth.json"
}
