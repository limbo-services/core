package svcauth

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/gogo/protobuf/gogoproto"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	pb "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
	plugin "github.com/gogo/protobuf/protoc-gen-gogo/plugin"

	. "github.com/fd/featherhead/tools/runtime/svcauth"
)

func init() {
	generator.RegisterPlugin(new(svcauth))
}

// grpc is an implementation of the Go protocol buffer compiler's
// plugin architecture.  It generates bindings for gRPC support.
type svcauth struct {
	gen *generator.Generator

	imports    generator.PluginImports
	contextPkg generator.Single
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

	imp := generator.NewPluginImports(g.gen)
	g.imports = imp
	g.contextPkg = imp.NewImport("golang.org/x/net/context")

	for i, service := range file.FileDescriptorProto.Service {
		g.generateService(file, service, i)
	}
}

// GenerateImports generates the import declaration for this file.
func (g *svcauth) GenerateImports(file *generator.FileDescriptor) {
	if len(file.FileDescriptorProto.Service) == 0 {
		return
	}

	g.imports.GenerateImports(file)
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
	g.P("authenticator ", innerServerType, "Auth")
	g.P("inner ", innerServerType)
	g.P("}")
	g.P()

	g.P("var _ ", innerServerType, " = (*", serverType, ")(nil)")
	g.P()

	g.P("func New", servName, "ServerAuthGuard(inner ", innerServerType, `, auth `, innerServerType, `Auth) `, innerServerType, " {")
	g.P("return &", serverType, "{inner: inner, authenticator: auth}")
	g.P("}")
	g.P()

	g.P(`type `, innerServerType, `Auth interface {`)
	var seenCallerTypes = map[string]bool{}
	for _, authMethod := range methods {
		method, authnInfo := authMethod.method, authMethod.Authn

		inputType, _ := g.gen.ObjectNamed(method.GetInputType()).(*generator.Descriptor)
		var callerType, _ = g.lookupMessageType(inputType, authnInfo.Caller)

		if !seenCallerTypes[callerType] {
			seenCallerTypes[callerType] = true
			g.P(`AuthenticateAs`, callerType, `(ctx `, g.contextPkg.Use(), `.Context, strategies []string, caller *`, callerType, `) error`)
		}
	}
	g.P(``)
	var seenCallerContextTypes = map[string]bool{}
	for _, authMethod := range methods {
		method, authzInfo := authMethod.method, authMethod.Authz
		if authzInfo == nil {
			continue
		}

		inputType, _ := g.gen.ObjectNamed(method.GetInputType()).(*generator.Descriptor)
		var callerType, _ = g.lookupMessageType(inputType, authzInfo.Caller)

		if authzInfo.Context != "" {
			var contextType, _ = g.lookupMessageType(inputType, authzInfo.Context)
			id := callerType + "/" + contextType
			if !seenCallerContextTypes[id] {
				seenCallerContextTypes[id] = true
				g.P(`Authorize`, callerType, `For`, contextType, `(ctx `, g.contextPkg.Use(), `.Context, scopes []string, caller *`, callerType, `, context *`, contextType, `) error`)
			}
		} else {
			id := callerType
			if !seenCallerContextTypes[id] {
				seenCallerContextTypes[id] = true
				g.P(`Authorize`, callerType, `(ctx `, g.contextPkg.Use(), `.Context, scopes []string, caller *`, callerType, `) error`)
			}
		}
	}
	g.P(`}`)
	g.P("")

	// Server handler implementations.
	var rules = map[string]int{}
	g.P("var (")
	for _, authMethod := range methods {
		authnInfo, authzInfo := authMethod.Authn, authMethod.Authz
		var rule string

		if authnInfo != nil && len(authnInfo.Strategies) > 0 {
			rule = fmt.Sprintf("%#v", authnInfo.Strategies)
		} else {
			rule = "[]string{}"
		}
		if id, found := rules[rule]; !found {
			id = len(rules) + 1
			g.P("authRule_", servName, "_", id, " = ", rule)
			rules[rule] = id
			authMethod.authnRuleID = id
		} else {
			authMethod.authnRuleID = id
		}

		if authzInfo != nil && len(authzInfo.Scopes) > 0 {
			rule = fmt.Sprintf("%#v", authzInfo.Scopes)
		} else {
			rule = "[]string{}"
		}
		if id, found := rules[rule]; !found {
			id = len(rules) + 1
			g.P("authRule_", servName, "_", id, " = ", rule)
			rules[rule] = id
			authMethod.authzRuleID = id
		} else {
			authMethod.authzRuleID = id
		}
	}
	g.P(")")
	g.P("")

	// Server handler implementations.
	for _, authMethod := range methods {
		method, authnInfo, authzInfo := authMethod.method, authMethod.Authn, authMethod.Authz

		inputType, _ := g.gen.ObjectNamed(method.GetInputType()).(*generator.Descriptor)
		var callerType, _ = g.lookupMessageType(inputType, authnInfo.Caller)

		g.P("func (s *", serverType, ") ", g.generateServerSignature(servName, method), " {")
		g.P("var (")
		if method.GetServerStreaming() || method.GetClientStreaming() {
			g.P("ctx = stream.Context()")
		}
		g.P(`caller `, callerType)
		if authzInfo != nil && authzInfo.Context != "" {
			var contextType, _ = g.lookupMessageType(inputType, authzInfo.Context)
			g.P(`context *`, contextType)
		}
		g.P(")")

		// Authenticate
		g.P("if err := s.authenticator.AuthenticateAs", callerType, "(ctx, ", "authRule_", servName, "_", authMethod.authnRuleID, ", &caller); err != nil {")
		if !method.GetServerStreaming() && !method.GetClientStreaming() {
			g.P("return nil, err")
		} else {
			g.P("return err")
		}
		g.P("}")

		if authzInfo != nil {
			var methodName string
			var args string
			if authzInfo.Context != "" {
				var contextType, _ = g.lookupMessageType(inputType, authzInfo.Context)
				methodName = `Authorize` + callerType + `For` + contextType
				args = "&caller, context"
				g.getMessage(inputType, authzInfo.Context, "input", "context", true)
			} else {
				methodName = `Authorize` + callerType
				args = "&caller"
			}
			g.P("if err := s.authenticator.", methodName, "(ctx, ", "authRule_", servName, "_", authMethod.authzRuleID, ", ", args, "); err != nil {")
			if !method.GetServerStreaming() && !method.GetClientStreaming() {
				g.P("return nil, err")
			} else {
				g.P("return err")
			}
			g.P("}")
			g.P("")
		}

		g.P("return s.inner.", g.generateServerCall(servName, method))
		g.P("}")
		g.P()
	}

	{
		var desc = AuthDescriptionSet{}

		for _, m := range methods {
			desc.Methods = append(desc.Methods, &AuthDescription{
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
		reqArgs = append(reqArgs, "ctx "+g.contextPkg.Use()+".Context")
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

type authMethod struct {
	file        *generator.FileDescriptor
	service     *pb.ServiceDescriptorProto
	method      *pb.MethodDescriptorProto
	authnRuleID int
	authzRuleID int
	Name        string     `json:"method"`
	Authn       *AuthnRule `json:"authn"`
	Authz       *AuthzRule `json:"authz"`
}

func (g *svcauth) findMethods(file *generator.FileDescriptor, service *pb.ServiceDescriptorProto) []*authMethod {
	methods := make([]*authMethod, 0, len(service.Method))

	var (
		defaultAuthnInfo *AuthnRule
		defaultAuthzInfo *AuthzRule
	)

	if service.Options != nil {
		v, _ := proto.GetExtension(service.Options, E_DefaultAuthn)
		defaultAuthnInfo, _ = v.(*AuthnRule)
	}

	if service.Options != nil {
		v, _ := proto.GetExtension(service.Options, E_DefaultAuthz)
		defaultAuthzInfo, _ = v.(*AuthzRule)
	}

	for _, method := range service.Method {
		var (
			authnInfo *AuthnRule
			authzInfo *AuthzRule
		)

		{ // authn
			v, _ := proto.GetExtension(method.Options, E_Authn)
			authnInfo, _ = v.(*AuthnRule)
			if authnInfo == nil {
				authnInfo = &AuthnRule{}
			}
			authnInfo = defaultAuthnInfo.Inherit(authnInfo)
			authnInfo.SetDefaults()
		}

		{ // authz
			v, _ := proto.GetExtension(method.Options, E_Authz)
			authzInfo, _ = v.(*AuthzRule)
			authzInfo = defaultAuthzInfo.Inherit(authzInfo)
			authzInfo.SetDefaults()
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

func (g *svcauth) lookupMessageType(inputType *generator.Descriptor, path string) (typeName string, isPtr bool) {
	partType := inputType
	parts := strings.Split(path, ".")
	lastIdx := len(parts) - 1
	for i, part := range parts {
		field := partType.GetFieldDescriptor(part)
		if field == nil {
			g.gen.Fail("unknown field", part, "in path", partType.GetName())
		}
		if !field.IsMessage() {
			g.gen.Fail("expected a message")
		}

		if lastIdx == i {
			if gogoproto.IsNullable(field) {
				return g.typeName(field.GetTypeName()), true
			} else {
				return g.typeName(field.GetTypeName()), false
			}
		}

		partType = g.gen.ObjectNamed(field.GetTypeName()).(*generator.Descriptor)
	}

	panic("unreachable")
}

func (g *svcauth) getMessage(inputType *generator.Descriptor, path, input, output string, inputIsNullable bool) {
	var (
		checks []string
		goPath string
	)

	goPath = input
	if inputIsNullable {
		checks = append(checks, input+" != nil")
	}

	for path != "" {

		// split path
		part := path
		idx := strings.IndexByte(path, '.')
		if idx >= 0 {
			part = path[:idx]
			path = path[idx+1:]
		} else {
			path = ""
		}

		// Get Field
		field := inputType.GetFieldDescriptor(part)
		if field == nil {
			g.gen.Fail("unknown field", part, "in message", inputType.GetName())
		}
		if !field.IsMessage() {
			g.gen.Fail("expected a message")
		}

		// Append code
		fieldGoName := g.gen.GetFieldName(inputType, field)
		goPath += "." + fieldGoName
		if gogoproto.IsNullable(field) {
			checks = append(checks, goPath+" != nil")
		}

		switch x := g.gen.ObjectNamed(field.GetTypeName()).(type) {
		case *generator.Descriptor:
			inputType = x
		case *generator.ImportedDescriptor:
			// f := g.gen.FileOf(x.File())
			f := x.File()
			fmt.Fprintf(os.Stderr, "x.TypeName    => %s\n", x.TypeName())
			fmt.Fprintf(os.Stderr, "x.PackageName => %s\n", x.PackageName())
			fmt.Fprintf(os.Stderr, "x.File        => %s\n", f.GetName())
			panic("invalid")
		default:
			panic("invalid")
		}
	}

	if len(checks) > 0 {
		g.P(`if `, strings.Join(checks, " && "), `{`)
		g.P(output, ` = `, goPath)
		g.P(`}`)
	} else {
		g.P(output, ` = `, goPath)
	}
}
