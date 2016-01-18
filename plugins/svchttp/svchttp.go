package svchttp

import (
	"encoding/json"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/fd/featherhead/pkg/api/httpapi/router"
	runtime "github.com/fd/featherhead/tools/runtime/svchttp"
	"github.com/gogo/protobuf/gogoproto"
	"github.com/gogo/protobuf/proto"
	pb "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
	plugin "github.com/gogo/protobuf/protoc-gen-gogo/plugin"
)

// Paths for packages used by code generated in this file,
// relative to the import_prefix of the generator.Generator.
const (
	contextPkgPath    = "golang.org/x/net/context"
	grpcPkgPath       = "google.golang.org/grpc"
	grpcCodesPkgPath  = "google.golang.org/grpc/codes"
	httpPkgPath       = "net/http"
	routerPkgPath     = "github.com/fd/featherhead/pkg/api/httpapi/router"
	runtimePkgPath    = "github.com/fd/featherhead/tools/runtime/svchttp"
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
	g.contextPkg = imp.NewImport(contextPkgPath)
	g.grpcCodesPkg = imp.NewImport(grpcCodesPkgPath)
	g.grpcPkg = imp.NewImport(grpcPkgPath)
	g.runtimePkg = imp.NewImport(runtimePkgPath)
	g.httpPkg = imp.NewImport(httpPkgPath)
	g.jsonPkg = imp.NewImport(jsonPkgPath)
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
	method        *pb.MethodDescriptorProto
	desc          *runtime.HttpRule
	descIndexPath string
}

func filterAPIs(methods []*pb.MethodDescriptorProto, svcIndex int) []*API {
	var apis = make([]*API, 0, len(methods))
	path := fmt.Sprintf("6,%d", svcIndex) // 6 means service.

	for i, method := range methods {
		v, _ := proto.GetExtension(method.Options, runtime.E_Http)
		info, _ := v.(*runtime.HttpRule)
		if info != nil {
			apis = append(apis, &API{
				method:        method,
				desc:          info,
				descIndexPath: fmt.Sprintf("%s,2,%d", path, i), // 2 means method in a service.
			})
		}
	}

	return apis
}

func (api *API) GetMethodAndPattern() (method, pattern string, ok bool) {

	switch x := api.desc.GetPattern().(type) {
	case *runtime.HttpRule_Get:
		method = "GET"
		pattern = x.Get
	case *runtime.HttpRule_Post:
		method = "POST"
		pattern = x.Post
	case *runtime.HttpRule_Put:
		method = "PUT"
		pattern = x.Put
	case *runtime.HttpRule_Patch:
		method = "PATCH"
		pattern = x.Patch
	case *runtime.HttpRule_Delete:
		method = "DELETE"
		pattern = x.Delete
	default:
		return "", "", false
	}

	return method, pattern, true
}

// generateService generates all the code for the named service.
func (g *svchttp) generateService(file *generator.FileDescriptor, service *pb.ServiceDescriptorProto, index int) {
	apis := filterAPIs(service.Method, index)
	if len(apis) == 0 {
		return
	}

	g.generateSwaggerSpec(file, service, apis)

	origServName := service.GetName()
	servName := generator.CamelCase(origServName)

	innerServerType := servName + "Server"
	handlerName := unexport(servName) + "Handler"

	g.P("type ", handlerName, " struct {")
	g.P("ss ", innerServerType)
	g.P("}")
	g.P()

	// Register handler
	var (
		routerRouter = g.routerPkg.Use() + ".Router"
	)
	g.P(`func Register`, servName, `Gateway(router *`, routerRouter, `, ss `, innerServerType, `) {`)
	g.P(`h := &`, handlerName, `{ss: ss}`)
	for _, api := range apis {
		_, method := api.desc, api.method

		httpMethod, pattern, ok := api.GetMethodAndPattern()
		if !ok {
			g.gen.Fail("xyz.featherhead.http requires a method: pattern")
		}

		handlerMethod := g.generateServerCallName(servName, method)
		g.P(`router.Addf(`, strconv.Quote(httpMethod), `,`, strconv.Quote(pattern), `, h. `, handlerMethod, `)`)
	}
	g.P("}")
	g.P()

	// Server handler implementations.
	for _, api := range apis {
		info, method := api.desc, api.method

		inputTypeName := method.GetInputType()
		inputType, _ := g.gen.ObjectNamed(inputTypeName).(*generator.Descriptor)

		outputTypeName := method.GetOutputType()
		outputType, _ := g.gen.ObjectNamed(outputTypeName).(*generator.Descriptor)

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
		g.P("func (h* ", handlerName, " )", handlerMethod, "(ctx ", contextContext, ", rw ", httpResponseWriter, ", req *", httpRequest, ") error {")
		g.P("// info:      ", info.String())
		g.P("//   method:  ", httpMethod)
		g.P("//   pattern: ", pattern)
		g.P("//   input:   ", inputType.String())
		g.P("//   output:  ", outputType.String())
		g.P("if req.Method != ", strconv.Quote(httpMethod), "{")
		g.P(`  return `, jujuErrors, `.MethodNotAllowedf("expected `, httpMethod, ` request")`)
		g.P("}")
		g.P()

		g.P(`var (`)
		g.P(`input `, g.typeName(inputTypeName))
		if len(vars) > 0 {
			routerP := g.routerPkg.Use() + ".P"
			g.P(`params = `, routerP, `(ctx)`)
		}
		g.P(`)`)
		g.P()

		g.P(`ctx = `, g.runtimePkg.Use(), `.AnnotateContext(ctx, req)`)
		g.P()

		if httpMethod == "POST" || httpMethod == "PUT" || httpMethod == "PATCH" {
			g.P("{ // from body")
			g.P("err := ", g.jsonPkg.Use(), ".NewDecoder(req.Body).Decode(&input)")
			g.P(`if err != nil {`)
			g.P(`return `, g.jujuErrorsPkg.Use(), `.Trace(err)`)
			g.P(`}`)
			g.P(`}`)
			g.P()
		}

		for param, value := range queryParams {
			g.P("{ // populate ", param, "=", value)
			g.P("var msg0 = &input")

			partType := inputType
			parts := strings.Split(value, ".")
			lastIdx := len(parts) - 1
			for i, part := range parts {
				field := partType.GetFieldDescriptor(part)
				fieldGoName := g.gen.GetFieldName(partType, field)

				if i == lastIdx {
					partType = nil
					if !field.IsString() {
						g.gen.Fail("expected a string")
					}

					g.P("msg", i, ".", fieldGoName, " = req.URL.Query().Get(", strconv.Quote(param), ")")

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

		for _, v := range vars {
			g.P("{ // populate ", v.Name)
			g.P("var msg0 = &input")

			partType := inputType
			parts := strings.Split(v.Name, ".")
			lastIdx := len(parts) - 1
			for i, part := range parts {
				field := partType.GetFieldDescriptor(part)
				fieldGoName := g.gen.GetFieldName(partType, field)

				if i == lastIdx {
					partType = nil
					if !field.IsString() {
						g.gen.Fail("expected a string")
					}

					g.P("msg", i, ".", fieldGoName, " = params.Get(", strconv.Quote(v.Name), ")")

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

		g.P(`{ // call`)
		if !method.GetServerStreaming() && !method.GetClientStreaming() {
			g.P(`output, err := h.ss.`, g.generateServerCall(servName, method))
			g.P(`if err != nil {`)
			g.P(`return `, g.jujuErrorsPkg.Use(), `.Trace(err)`)
			g.P(`}`)
			g.P(g.runtimePkg.Use(), `.RenderMessageJSON(rw, 200, output)`)
			g.P("return nil")
		} else {
			if info.Paged {
				g.P(`ss, err := `, g.runtimePkg.Use(), `.NewPagedServerStream(ctx, rw, `, int(info.PageSize), `)`)
				g.P(`if err != nil {`)
				g.P(`return `, g.jujuErrorsPkg.Use(), `.Trace(err)`)
				g.P(`}`)
				g.P(`defer ss.Close()`)
				g.P(`stream := &`, unexport(servName), method.GetName(), `Server{ss}`)
				g.P(`err = h.ss.`, g.generateServerCall(servName, method))
			} else {
				g.gen.Fail("unsupported stream response")
			}
		}
		g.P(`}`)
		g.P()

		g.P("return nil")
		g.P("}")
		g.P()

	}

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

func (g *svchttp) generateSwaggerSpec(file *generator.FileDescriptor, service *pb.ServiceDescriptorProto, apis []*API) {

	type Info struct {
		Title          string `json:"title"`
		Description    string `json:"description,omitempty"`
		TermsOfService string `json:"termsOfService,omitempty"`
		Version        string `json:"version"`
	}

	type ParameterObject struct {
	}

	type ResponseObject struct {
		Description string `json:"description"`
		// schema      string
		// headers
		// examples
	}

	type OperationObject struct {
		Summary     string                    `json:"summary,omitempty"`
		Description string                    `json:"description,omitempty"`
		Parameters  []ParameterObject         `json:"parameters,omitempty"`
		Responses   map[string]ResponseObject `json:"responses"`
	}

	type Swagger struct {
		Swagger string `json:"swagger"`
		Info    Info   `json:"info"`
		// host
		// basePath
		Schemes  []string `json:"schemes"`  // ["https", "wss"]
		Consumes []string `json:"consumes"` // ["application/json; charset=utf-8"]
		Produces []string `json:"produces"` // ["application/json; charset=utf-8"]

		// swagger.Paths["<path>"]
		Paths map[string]map[string]*OperationObject `json:"paths"`
	}

	spec := Swagger{
		Swagger: "2.0",
		Info: Info{
			Title:   "Featherhead - " + service.GetName(),
			Version: "1",
		},

		Schemes:  []string{"https", "wss"},
		Consumes: []string{"application/json; charset=utf-8"},
		Produces: []string{"application/json; charset=utf-8"},

		Paths: map[string]map[string]*OperationObject{},
	}

	for _, api := range apis {
		method, pattern, ok := api.GetMethodAndPattern()
		if !ok {
			continue
		}

		pathItem := spec.Paths[pattern]
		if pathItem == nil {
			pathItem = make(map[string]*OperationObject)
			spec.Paths[pattern] = pathItem
		}

		operation := pathItem[strings.ToLower(method)]
		if operation == nil {
			operation = &OperationObject{}
			pathItem[strings.ToLower(method)] = operation
		}

		if operation.Responses == nil {
			operation.Responses = make(map[string]ResponseObject)
		}

		operation.Description = strings.TrimSpace(g.gen.Comments(api.descIndexPath))
		operation.Responses["200"] = ResponseObject{
			Description: "Response on success",
		}
	}

	data, err := json.Marshal(&spec)
	if err != nil {
		g.gen.Error(err)
	}

	g.gen.Response.File = append(g.gen.Response.File, &plugin.CodeGeneratorResponse_File{
		Name:    proto.String(swaggerSpecFileName(*file.Name)),
		Content: proto.String(string(data)),
	})
}

func swaggerSpecFileName(name string) string {
	ext := path.Ext(name)
	if ext == ".proto" || ext == ".protodevel" {
		name = name[0 : len(name)-len(ext)]
	}
	return name + ".swagger.json"
}
