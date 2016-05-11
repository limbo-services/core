package svchttp

import (
	"fmt"
	"strconv"
	"strings"

	"limbo.services/protobuf/gogoproto"
	"limbo.services/protobuf/proto"
	pb "limbo.services/protobuf/protoc-gen-gogo/descriptor"
	"limbo.services/protobuf/protoc-gen-gogo/generator"

	. "limbo.services/core/runtime/limbo"
	"limbo.services/router"
)

// Paths for packages used by code generated in this file,
// relative to the import_prefix of the generator.Generator.
const (
	contextPkgPath    = "golang.org/x/net/context"
	grpcPkgPath       = "google.golang.org/grpc"
	grpcCodesPkgPath  = "google.golang.org/grpc/codes"
	httpPkgPath       = "net/http"
	routerPkgPath     = "limbo.services/router"
	runtimePkgPath    = "limbo.services/core/runtime/limbo"
	jujuErrorsPkgPath = "github.com/juju/errors"
	jsonPkgPath       = "encoding/json"
)

func init() {
	generator.RegisterPlugin(new(svchttp))
}

// grpc is an implementation of the Go protocol buffer compiler's
// plugin architecture.  It generates bindings for gRPC support.
type svchttp struct {
	gen *generator.Generator

	imports       generator.PluginImports
	strconvPkg    generator.Single
	httpPkg       generator.Single
	grpcPkg       generator.Single
	contextPkg    generator.Single
	grpcCodesPkg  generator.Single
	routerPkg     generator.Single
	jujuErrorsPkg generator.Single
	jsonPkg       generator.Single
	runtimePkg    generator.Single
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

	imp := generator.NewPluginImports(g.gen)
	g.imports = imp
	g.strconvPkg = imp.NewImport("strconv")
	g.contextPkg = imp.NewImport(contextPkgPath)
	g.grpcCodesPkg = imp.NewImport(grpcCodesPkgPath)
	g.grpcPkg = imp.NewImport(grpcPkgPath)
	g.httpPkg = imp.NewImport(httpPkgPath)
	g.jsonPkg = imp.NewImport(jsonPkgPath)
	g.runtimePkg = imp.NewImport(runtimePkgPath)
	g.jujuErrorsPkg = imp.NewImport(jujuErrorsPkgPath)
	g.routerPkg = imp.NewImport(routerPkgPath)

	for i, service := range file.FileDescriptorProto.Service {
		g.generateService(file, service, i)
	}
}

// GenerateImports generates the import declaration for this file.
func (g *svchttp) GenerateImports(file *generator.FileDescriptor) {
	if len(file.FileDescriptorProto.Service) == 0 {
		return
	}

	g.imports.GenerateImports(file)
}

func unexport(s string) string { return strings.ToLower(s[:1]) + s[1:] }

type API struct {
	service       *pb.ServiceDescriptorProto
	method        *pb.MethodDescriptorProto
	desc          *HttpRule
	descIndexPath string

	descName string
	stream   bool
	index    int
}

func filterAPIs(service *pb.ServiceDescriptorProto, methods []*pb.MethodDescriptorProto, svcIndex int) []*API {
	var apis = make([]*API, 0, len(methods))
	path := fmt.Sprintf("6,%d", svcIndex) // 6 means service.

	var (
		descName  = "_" + service.GetName() + "_serviceDesc"
		methodIdx = 0
		streamIdx = 0
	)

	for i, method := range methods {
		stream := method.GetClientStreaming() || method.GetServerStreaming()
		index := 0
		if stream {
			index = streamIdx
		} else {
			index = methodIdx
		}

		v, _ := proto.GetExtension(method.Options, E_Http)
		info, _ := v.(*HttpRule)
		if info != nil {
			apis = append(apis, &API{
				service:       service,
				method:        method,
				desc:          info,
				descIndexPath: fmt.Sprintf("%s,2,%d", path, i), // 2 means method in a service.
				descName:      descName,
				stream:        stream,
				index:         index,
			})
		}

		if stream {
			streamIdx++
		} else {
			methodIdx++
		}
	}

	return apis
}

func (api *API) GetMethodAndPattern() (method, pattern string, ok bool) {

	switch x := api.desc.GetPattern().(type) {
	case *HttpRule_Get:
		method = "GET"
		pattern = x.Get
	case *HttpRule_Post:
		method = "POST"
		pattern = x.Post
	case *HttpRule_Put:
		method = "PUT"
		pattern = x.Put
	case *HttpRule_Patch:
		method = "PATCH"
		pattern = x.Patch
	case *HttpRule_Delete:
		method = "DELETE"
		pattern = x.Delete
	default:
		return "", "", false
	}

	return method, pattern, true
}

// generateService generates all the code for the named service.
func (g *svchttp) generateService(file *generator.FileDescriptor, service *pb.ServiceDescriptorProto, index int) {
	apis := filterAPIs(service, service.Method, index)
	if len(apis) == 0 {
		return
	}

	origServName := service.GetName()
	fullServName := file.GetPackage() + "." + origServName
	servName := generator.CamelCase(origServName)
	gatewayVarName := "_" + servName + "_gatewayDesc"

	g.gen.AddInitf("%s.RegisterGatewayDesc(&%s)", g.runtimePkg.Use(), gatewayVarName)

	g.P(`var `, gatewayVarName, ` = `, g.runtimePkg.Use(), `.GatewayDesc{`)
	g.P(`ServiceName: `, strconv.Quote(fullServName), `,`)
	g.P(`HandlerType: ((*`, servName, `Server)(nil)),`)
	g.P(`Routes: []`, g.runtimePkg.Use(), `.RouteDesc{`)
	for _, api := range apis {
		_, method := api.desc, api.method

		httpMethod, pattern, ok := api.GetMethodAndPattern()
		if !ok {
			g.gen.Fail("xyz.featherhead.http requires a method: pattern")
		}

		if idx := strings.IndexRune(pattern, '?'); idx >= 0 {
			pattern = pattern[:idx]
		}

		g.P(`{`)
		g.P(`Method: `, strconv.Quote(httpMethod), `,`)
		g.P(`Pattern: `, strconv.Quote(pattern), `,`)
		g.P(`Handler: `, g.generateServerCallName(servName, method), `,`)
		g.P("},")
	}
	g.P("},")
	g.P("}")
	g.P()

	// Server handler implementations.
	for _, api := range apis {
		info, method := api.desc, api.method

		inputTypeName := method.GetInputType()
		inputType, _ := g.gen.ObjectNamed(inputTypeName).(*generator.Descriptor)

		httpMethod, pattern, ok := api.GetMethodAndPattern()
		queryParams := map[string]string{}
		if !ok {
			g.gen.Fail("xyz.featherhead.http requires a method: pattern")
		}

		if idx := strings.IndexRune(pattern, '?'); idx >= 0 {
			queryString := pattern[idx+1:]
			pattern = pattern[:idx]

			for _, pair := range strings.SplitN(queryString, "&", -1) {
				idx := strings.Index(pair, "={")
				if pair[len(pair)-1] != '}' || idx < 0 {
					g.gen.Fail("invalid query paramter")
				}
				queryParams[pair[:idx]] = pair[idx+2 : len(pair)-1]
			}
		}

		vars, err := router.ExtractVariables(pattern)
		if err != nil {
			g.gen.Error(err)
			return
		}

		var (
			httpResponseWriter = g.httpPkg.Use() + ".ResponseWriter"
			httpRequest        = g.httpPkg.Use() + ".Request"
			contextContext     = g.contextPkg.Use() + ".Context"
		)

		handlerMethod := g.generateServerCallName(servName, method)
		jujuErrors := g.jujuErrorsPkg.Use()
		g.P("func ", handlerMethod, "(srvDesc *", g.grpcPkg.Use(), ".ServiceDesc, srv interface{}, ctx ", contextContext, ", rw ", httpResponseWriter, ", req *", httpRequest, ") error {")
		g.P("if req.Method != ", strconv.Quote(httpMethod), "{")
		g.P(`  return `, jujuErrors, `.MethodNotAllowedf("expected `, httpMethod, ` request")`)
		g.P("}")
		g.P()

		if len(vars) > 0 {
			routerP := g.routerPkg.Use() + ".P"
			g.P(`params := `, routerP, `(ctx)`)
		}

		g.P(`stream, err := `, g.runtimePkg.Use(), `.NewServerStream(ctx, rw, req, `,
			method.GetServerStreaming(), `, `, method.GetClientStreaming(), `, `, int(info.PageSize), `, func(x interface{}) error {`)
		g.P(`input := x.(*`, g.typeName(inputTypeName), `)`)
		g.P(`_ = input`)
		g.P()

		for param, value := range queryParams {
			g.P("// populate ?", param, "=", value)
			g.generateHttpMapping(inputType, value, "req.URL.Query().Get("+strconv.Quote(param)+")")
		}

		for _, v := range vars {
			g.P("// populate {", v.Name, "}")
			g.generateHttpMapping(inputType, v.Name, "params.Get("+strconv.Quote(v.Name)+")")
		}

		g.P(`return nil`)
		g.P(`})`)
		g.P()

		if !api.stream {
			g.P(`desc := &srvDesc.Methods[`, api.index, `]`)
			g.P(`output, err := desc.Handler(srv, stream.Context(), stream.RecvMsg)`)
			g.P(`if err == nil && output == nil {`)
			g.P(`err = `, g.grpcPkg.Use(), `.Errorf(`, g.grpcCodesPkg.Use(), `.Internal, "internal server error")`)
			g.P(`}`)
			g.P(`if err == nil {`)
			g.P(`err = stream.SendMsg(output)`)
			g.P(`}`)
		} else {
			g.P(`desc := &srvDesc.Streams[`, api.index, `]`)
			g.P(`err = desc.Handler(srv, stream)`)
		}
		g.P(`if err != nil {`)
		g.P(`stream.SetError(err)`)
		g.P(`}`)
		g.P()

		g.P(`return stream.CloseSend()`)
		g.P("}")
		g.P()

	}

}

func (g *svchttp) generateHttpMapping(inputType *generator.Descriptor, path string, value string) {
	g.P("{")
	g.P("var msg0 = input")

	partType := inputType
	parts := strings.Split(path, ".")
	lastIdx := len(parts) - 1
	for i, part := range parts {
		field := partType.GetFieldDescriptor(part)
		if field == nil {
			g.gen.Fail("unknown field ", part)
		}
		fieldGoName := g.gen.GetFieldName(partType, field)

		if i == lastIdx {
			partType = nil

			g.P("val := ", value)
			if field.IsString() {
				g.P("msg", i, ".", fieldGoName, " = val")
			} else if field.IsBytes() {
				g.P("msg", i, ".", fieldGoName, " = []byte(val)")
			} else if *field.Type == pb.FieldDescriptorProto_TYPE_INT64 ||
				*field.Type == pb.FieldDescriptorProto_TYPE_SINT64 ||
				*field.Type == pb.FieldDescriptorProto_TYPE_SFIXED64 {
				g.P("intVal, _ := ", g.strconvPkg.Use(), ".ParseInt(val, 10, 64)")
				g.P("msg", i, ".", fieldGoName, " = int64(intVal)")
			} else if *field.Type == pb.FieldDescriptorProto_TYPE_INT32 ||
				*field.Type == pb.FieldDescriptorProto_TYPE_SINT32 ||
				*field.Type == pb.FieldDescriptorProto_TYPE_SFIXED32 {
				g.P("intVal, _ := ", g.strconvPkg.Use(), ".ParseInt(val, 10, 32)")
				g.P("msg", i, ".", fieldGoName, " = int32(intVal)")
			} else if *field.Type == pb.FieldDescriptorProto_TYPE_UINT64 ||
				*field.Type == pb.FieldDescriptorProto_TYPE_FIXED64 {
				g.P("intVal, _ := ", g.strconvPkg.Use(), ".ParseUint(val, 10, 64)")
				g.P("msg", i, ".", fieldGoName, " = uint64(intVal)")
			} else if *field.Type == pb.FieldDescriptorProto_TYPE_UINT32 ||
				*field.Type == pb.FieldDescriptorProto_TYPE_FIXED32 {
				g.P("intVal, _ := ", g.strconvPkg.Use(), ".ParseUint(val, 10, 32)")
				g.P("msg", i, ".", fieldGoName, " = uint32(intVal)")
			} else {
				g.gen.Fail("expected a string")
			}

		} else {
			if !field.IsMessage() {
				g.gen.Fail("expected a message")
			}

			if gogoproto.IsNullable(field) {
				g.P(`if msg`, i, `.`, fieldGoName, ` == nil {`)
				g.P("msg", i, ".", fieldGoName, " = &", g.typeName(field.GetTypeName()), "{}")
				g.P(`}`)
				g.P("var msg", i+1, " = msg", i, ".", fieldGoName)
			} else {
				g.P("var msg", i+1, " = &msg", i, ".", fieldGoName)
			}

			partType = g.gen.ObjectNamed(field.GetTypeName()).(*generator.Descriptor)
		}
	}

	g.P("}")
	g.P()
}

// generateServerSignature returns the server-side signature for a method.
func (g *svchttp) generateServerCallName(servName string, method *pb.MethodDescriptorProto) string {
	origMethName := method.GetName()
	methName := "_http_" + servName + "_" + generator.CamelCase(origMethName)
	if reservedClientName[methName] {
		methName += "_"
	}

	return methName
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
		reqArgs = append(reqArgs, "&input")
	}
	if method.GetServerStreaming() || method.GetClientStreaming() {
		reqArgs = append(reqArgs, "stream")
	}

	return methName + "(" + strings.Join(reqArgs, ", ") + ") "
}
