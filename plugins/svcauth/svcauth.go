package svcauth

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/limbo-services/protobuf/gogoproto"
	"github.com/limbo-services/protobuf/proto"
	pb "github.com/limbo-services/protobuf/protoc-gen-gogo/descriptor"
	"github.com/limbo-services/protobuf/protoc-gen-gogo/generator"

	. "github.com/limbo-services/core/runtime/limbo"
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
	runtimePkg generator.Single

	messages map[string]*generator.Descriptor
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
	g.messages = map[string]*generator.Descriptor{}
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
	for _, msg := range file.Messages() {
		name := file.GetPackage() + "." + msg.GetName()
		g.messages[name] = msg
	}

	if len(file.FileDescriptorProto.Service) == 0 {
		return
	}

	imp := generator.NewPluginImports(g.gen)
	g.imports = imp
	g.contextPkg = imp.NewImport("golang.org/x/net/context")
	g.runtimePkg = imp.NewImport("github.com/limbo-services/core/runtime/limbo")

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
	fullServName := file.GetPackage() + "." + origServName
	servName := generator.CamelCase(origServName)
	authDescVarName := "_" + servName + "_authDesc"

	methodsByName := make(map[*pb.MethodDescriptorProto]*authMethod)
	for _, m := range methods {
		methodsByName[m.method] = m
	}

	g.gen.AddInitf("%s.RegisterServiceAuthDesc(&%s)", g.runtimePkg.Use(), authDescVarName)

	var interfaceMethods []string

	g.P(`var `, authDescVarName, ` = `, g.runtimePkg.Use(), `.ServiceAuthDesc{`)
	g.P(`ServiceName: `, strconv.Quote(fullServName), `,`)
	g.P(`HandlerType: ((*`, servName, `Server)(nil)),`)
	g.P(`AuthHandlerType: ((*`, servName, `ServerAuth)(nil)),`)
	g.P(`Methods: []`, g.runtimePkg.Use(), `.MethodAuthDesc{`)
	for _, method := range service.Method {
		if method.GetServerStreaming() || method.GetClientStreaming() {
			continue
		}
		g.P(`{`)
		g.P(`MethodName: `, strconv.Quote(method.GetName()), `,`)
		g.generateDesc(servName, method, methodsByName[method], &interfaceMethods)
		g.P("},")
	}
	g.P("},")
	g.P(`Streams: []`, g.runtimePkg.Use(), `.StreamAuthDesc{`)
	for _, method := range service.Method {
		if !method.GetServerStreaming() && !method.GetClientStreaming() {
			continue
		}
		g.P(`{`)
		g.P(`StreamName: `, strconv.Quote(method.GetName()), `,`)
		g.generateDesc(servName, method, methodsByName[method], &interfaceMethods)
		g.P("},")
	}
	g.P("},")
	g.P("}")
	g.P()

	if len(interfaceMethods) > 0 {
		sort.Strings(interfaceMethods)
		last := ""
		g.P(`type `, servName, `ServerAuth interface {`)
		for _, sig := range interfaceMethods {
			if sig != last {
				last = sig
				g.P(sig)
			}
		}
		g.P(`}`)
	}
}

func (g *svcauth) generateDesc(servName string, method *pb.MethodDescriptorProto, authMethod *authMethod, interfaceMethods *[]string) {
	emittedCaller := false
	if authMethod != nil && authMethod.Authn != nil {
		var (
			inputType, _    = g.gen.ObjectNamed(method.GetInputType()).(*generator.Descriptor)
			callerType      = g.lookupMessageType(inputType, authMethod.Authn.Caller)
			shortCallerType = toShort(callerType)
		)

		g.P(`Strategies: []string{`)
		for _, s := range authMethod.Authn.Strategies {
			g.P(strconv.Quote(s), `,`)
		}
		g.P(`},`)

		emittedCaller = true
		g.P(`CallerType: ((*`, callerType, `)(nil)),`)
		g.P(`GetCaller: func(msg interface{}) interface{} {`)
		g.P(`var (`)
		g.P(`input = msg.(*`, g.typeName(method.GetInputType()), `)`)
		g.P(`caller *`, callerType)
		g.P(`)`)
		g.getMessage(inputType, authMethod.Authn.Caller, "input", "caller", true)
		g.P(`if caller == nil {`)
		g.P(`caller = &`, callerType, `{}`)
		g.P(`}`)
		g.P(`return caller`)
		g.P(`},`)
		g.P(`SetCaller: func(msg interface{}, v interface{}) {`)
		g.P(`var (`)
		g.P(`input = msg.(*`, g.typeName(method.GetInputType()), `)`)
		g.P(`caller = v.(*`, callerType, `)`)
		g.P(`)`)
		g.setMessage(inputType, authMethod.Authn.Caller, "input", "caller", true)
		g.P(`},`)
		g.P(`Authenticate: func(h interface{}, ctx context.Context, strategies []string, c interface{}) error {`)
		g.P(`return h.(`, servName, `ServerAuth).AuthenticateAs`, shortCallerType, `(ctx, strategies, c.(*`, callerType, `))`)
		g.P(`},`)

		*interfaceMethods = append(*interfaceMethods,
			fmt.Sprintf("AuthenticateAs%s(%s.Context, []string, *%s) error", shortCallerType, g.contextPkg.Use(), callerType))
	}
	if authMethod != nil && authMethod.Authz != nil {
		var (
			inputType, _    = g.gen.ObjectNamed(method.GetInputType()).(*generator.Descriptor)
			callerType      = g.lookupMessageType(inputType, authMethod.Authz.Caller)
			shortCallerType = toShort(callerType)
			// AuthorizeCallerForCertificateRef(ctx golang_org_x_net_context.Context, scope string, caller *Caller, context *CertificateRef) error
			authzCallName = "Authorize" + shortCallerType
			authzCallArgs = "ctx, scope, caller.(*" + callerType + ")"
			authzDeclArgs = g.contextPkg.Use() + ".Context, string, *" + callerType
		)

		if !emittedCaller {
			g.P(`CallerType: ((*`, callerType, `)(nil)),`)
			g.P(`GetCaller: func(msg interface{}) interface{} {`)
			g.P(`var (`)
			g.P(`input = msg.(*`, g.typeName(method.GetInputType()), `)`)
			g.P(`caller *`, callerType)
			g.P(`)`)
			g.getMessage(inputType, authMethod.Authz.Caller, "input", "caller", true)
			g.P(`if caller == nil {`)
			g.P(`caller = &`, callerType, `{}`)
			g.P(`}`)
			g.P(`return caller`)
			g.P(`},`)
		}

		g.P(`Scope: `, strconv.Quote(authMethod.Authz.Scope), `,`)

		if authMethod.Authz.Context != "" {
			var (
				contextType      = g.lookupMessageType(inputType, authMethod.Authz.Context)
				shortContextType = toShort(contextType)
			)
			authzCallName += "For" + shortContextType
			authzCallArgs += ", context.(*" + contextType + ")"
			authzDeclArgs += ", *" + contextType

			g.P(`ContextType: ((*`, contextType, `)(nil)),`)
			g.P(`GetContext: func(msg interface{}) interface{} {`)
			g.P(`var (`)
			g.P(`input = msg.(*`, g.typeName(method.GetInputType()), `)`)
			g.P(`context *`, contextType)
			g.P(`)`)
			g.getMessage(inputType, authMethod.Authz.Context, "input", "context", true)
			g.P(`return context`)
			g.P(`},`)
		}

		g.P(`Authorize: func(h interface{}, ctx context.Context, scope string, caller, context interface{}) error {`)
		g.P(`return h.(`, servName, `ServerAuth).`, authzCallName, `(`, authzCallArgs, `)`)
		g.P(`},`)

		*interfaceMethods = append(*interfaceMethods,
			fmt.Sprintf("%s(%s) error", authzCallName, authzDeclArgs))
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
			if authnInfo == nil && defaultAuthnInfo != nil {
				authnInfo = &AuthnRule{}
			}
			if authnInfo != nil {
				authnInfo = defaultAuthnInfo.Inherit(authnInfo)
				authnInfo.SetDefaults()
			}
		}

		{ // authz
			v, _ := proto.GetExtension(method.Options, E_Authz)
			authzInfo, _ = v.(*AuthzRule)
			if authzInfo == nil && defaultAuthzInfo != nil {
				authzInfo = &AuthzRule{}
			}
			if authzInfo != nil {
				authzInfo = defaultAuthzInfo.Inherit(authzInfo)
				authzInfo.SetDefaults()
			}
		}

		if authnInfo == nil && authzInfo == nil {
			continue
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

func (g *svcauth) lookupMessageType(inputType *generator.Descriptor, path string) (typeName string) {
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
			return g.typeName(field.GetTypeName())
		}

		typeName := strings.TrimPrefix(field.GetTypeName(), ".")
		partType = g.messages[typeName]
	}

	panic("unreachable")
}

func (g *svcauth) getMessage(inputType *generator.Descriptor, path, input, output string, inputIsNullable bool) {
	var (
		checks     []string
		goPath     string
		isNullable = inputIsNullable
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
			g.gen.Fail("unknown field", strconv.Quote(part), "in message", inputType.GetName())
		}
		if !field.IsMessage() {
			g.gen.Fail("expected a message")
		}

		// Append code
		fieldGoName := g.gen.GetFieldName(inputType, field)
		goPath += "." + fieldGoName
		if gogoproto.IsNullable(field) {
			checks = append(checks, goPath+" != nil")
			isNullable = true
		} else {
			isNullable = false
		}

		inputType = g.messages[strings.TrimPrefix(field.GetTypeName(), ".")]
	}

	if len(checks) > 0 {
		g.P(`if `, strings.Join(checks, " && "), `{`)
		if isNullable {
			g.P(output, ` = `, goPath)
		} else {
			g.P(output, ` = &`, goPath)
		}
		g.P(`}`)
	} else {
		if isNullable {
			g.P(output, ` = `, goPath)
		} else {
			g.P(output, ` = &`, goPath)
		}
	}
}

func (g *svcauth) setMessage(inputType *generator.Descriptor, path, input, output string, inputIsNullable bool) {
	var (
		goPath           string
		outputIsNullable bool
	)

	goPath = input
	if inputIsNullable {
		g.P(`if `, goPath, `== nil {`)
		g.P(goPath, `= &`, g.gen.TypeName(inputType), `{}`)
		g.P(`}`)
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

		inputType = g.messages[strings.TrimPrefix(field.GetTypeName(), ".")]

		if gogoproto.IsNullable(field) && path != "" {
			g.P(`if `, goPath, `== nil {`)
			g.P(goPath, `= &`, g.gen.TypeName(inputType), `{}`)
			g.P(`}`)
		}

		if gogoproto.IsNullable(field) {
			outputIsNullable = true
		} else {
			outputIsNullable = false
		}
	}

	if outputIsNullable {
		g.P(goPath, ` = `, output)
	} else {
		g.P(goPath, ` = &`, output)
	}
}

type authzContext struct {
	CallerType       string
	ContextType      string
	ShortCallerType  string
	ShortContextType string
	Scopes           []string
}

func (g *svcauth) lookupAuthzContexts(methods []*authMethod) []*authzContext {
	var m = map[string]*authzContext{}

	for _, authMethod := range methods {
		var (
			inputType        *generator.Descriptor
			callerType       string
			contextType      string
			shortCallerType  string
			shortContextType string
			id               string
		)

		method, authzInfo := authMethod.method, authMethod.Authz
		if authzInfo == nil {
			continue
		}

		inputType, _ = g.gen.ObjectNamed(method.GetInputType()).(*generator.Descriptor)
		callerType = g.lookupMessageType(inputType, authzInfo.Caller)
		id = callerType

		if authzInfo.Context != "" {
			contextType = g.lookupMessageType(inputType, authzInfo.Context)
			id = callerType + "/" + contextType
		}

		shortCallerType = toShort(callerType)
		shortContextType = toShort(contextType)

		ctx := m[id]

		if ctx == nil {
			ctx = &authzContext{
				CallerType:       callerType,
				ContextType:      contextType,
				ShortCallerType:  shortCallerType,
				ShortContextType: shortContextType,
			}
			m[id] = ctx
		}

		ctx.Scopes = append(ctx.Scopes, authzInfo.Scope)
	}

	var l = make([]*authzContext, 0, len(m))
	var keys = make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		ctx := m[k]
		l = append(l, ctx)
		sort.Strings(ctx.Scopes)
		tmp := ctx.Scopes
		ctx.Scopes = ctx.Scopes[:0]
		lastScope := ""
		for _, scope := range tmp {
			if scope != lastScope {
				lastScope = scope
				ctx.Scopes = append(ctx.Scopes, scope)
			}
		}
	}

	return l
}

func toShort(s string) string {
	if idx := strings.LastIndexByte(s, '.'); idx >= 0 {
		return s[idx+1:]
	}
	return s
}
