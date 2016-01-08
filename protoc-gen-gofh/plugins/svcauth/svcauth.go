package svcauth

import (
	"strings"

	"github.com/gogo/protobuf/proto"
	pb "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"

	fhannotations "github.com/fd/featherhead/proto"
)

// Paths for packages used by code generated in this file,
// relative to the import_prefix of the generator.Generator.
const (
	contextPkgPath       = "golang.org/x/net/context"
	grpcPkgPath          = "google.golang.org/grpc"
	grpcCodesPkgPath     = "google.golang.org/grpc/codes"
	grpcMetadataPkgPath  = "google.golang.org/grpc/metadata"
	fhannotationsPkgPath = "github.com/fd/featherhead/proto"
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
	g.gen.PrintImport("fhannotations_svcauth", fhannotationsPkgPath)
	g.gen.PrintImport("metadata_svcauth", grpcMetadataPkgPath)
}

func unexport(s string) string { return strings.ToLower(s[:1]) + s[1:] }

// generateService generates all the code for the named service.
func (g *svcauth) generateService(file *generator.FileDescriptor, service *pb.ServiceDescriptorProto, index int) {

	origServName := service.GetName()
	servName := generator.CamelCase(origServName)

	// Server interface.
	innerServerType := servName + "Server"
	serverType := unexport(servName) + "ServerAuthGuard"
	g.P("type ", serverType, " struct {")
	g.P("authenticator fhannotations_svcauth.Authenticator")
	g.P("inner ", innerServerType)
	g.P("}")
	g.P()

	g.P("var _ ", innerServerType, " = (*", serverType, ")(nil)")
	g.P()

	g.P("func New", servName, "ServerAuthGuard(inner ", innerServerType, ") ", innerServerType, " {")
	g.P("return &", serverType, "{inner: inner}")
	g.P("}")
	g.P()

	// Server handler implementations.
	for _, method := range service.Method {
		var (
			authnInfo *fhannotations.AuthnRule
			authzInfo *fhannotations.AuthzRule
		)

		{ // authn
			v, _ := proto.GetExtension(method.Options, fhannotations.E_Authn)
			authnInfo, _ = v.(*fhannotations.AuthnRule)
			if authnInfo == nil {
				authnInfo = &fhannotations.AuthnRule{}
				authnInfo.Gateway = fhannotations.AuthnRule_DENY
			}
			authnInfo.SetDefaults()
		}

		{ // authz
			v, _ := proto.GetExtension(method.Options, fhannotations.E_Authz)
			authzInfo, _ = v.(*fhannotations.AuthzRule)
			if authzInfo != nil {
				authzInfo.SetDefaults()
			}
		}

		authnRuleName := g.generateAuthnRuleName(servName, method)
		authzRuleName := g.generateAuthzRuleName(servName, method)
		g.P("var (")
		g.P(authnRuleName, " *fhannotations_svcauth.AuthnRule = ", replaceFhAnnotationNames(authnInfo.GoString()))
		g.P(authzRuleName, " *fhannotations_svcauth.AuthzRule = ", replaceFhAnnotationNames(authzInfo.GoString()))
		g.P(")")
		g.P("")

		g.P("func (s *", serverType, ") ", g.generateServerSignature(servName, method), " {")

		g.P("var (")
		g.P("info fhannotations_svcauth.AuthInfo")
		if method.GetServerStreaming() || method.GetClientStreaming() {
			g.P("ctx = stream.Context()")
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

		g.P("md, ok := metadata_svcauth.FromContext(ctx)")
		g.P(`if !ok || md == nil {`)
		g.P(`panic("no metadata.MD in context.Context")`)
		g.P(`}`)
		g.P(`md["x-user-id"] = []string{info.UserID}`)
		g.P(`md["x-client-id"] = []string{info.ClientID}`)
		g.P(`md["x-subject-type"] = []string{info.SubjectType}`)
		g.P(`md["x-subject-id"] = []string{info.SubjectID}`)
		g.P("return s.inner.", g.generateServerCall(servName, method))
		g.P("}")
		g.P()
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
	return strings.Replace(s, "fhannotations.", "fhannotations_svcauth.", -1)
}
