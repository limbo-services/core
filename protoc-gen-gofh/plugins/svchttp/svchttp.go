package svchttp

import (
	"strconv"
	"strings"

	"github.com/fd/featherhead/pkg/api/httpapi/router"
	"github.com/fd/featherhead/proto"
	"github.com/gogo/protobuf/gogoproto"
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
	httpPkgPath      = "net/http"
	routerPkgPath    = "github.com/fd/featherhead/pkg/api/httpapi/router"
	errorsPkgPath    = "github.com/juju/errors"
	jsonPkgPath      = "encoding/json"
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
	g.P(`var _ *http_svchttp.Request`)
	g.P(`var _ *router_svchttp.Router`)
	g.P(`var _ = errors_svchttp.Errorf`)
	g.P(`var _ = json_svchttp.Marshal`)
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
	g.gen.PrintImport("http_svchttp", httpPkgPath)
	g.gen.PrintImport("router_svchttp", routerPkgPath)
	g.gen.PrintImport("errors_svchttp", errorsPkgPath)
	g.gen.PrintImport("json_svchttp", jsonPkgPath)
}

func unexport(s string) string { return strings.ToLower(s[:1]) + s[1:] }

// generateService generates all the code for the named service.
func (g *svchttp) generateService(file *generator.FileDescriptor, service *pb.ServiceDescriptorProto, index int) {

	origServName := service.GetName()
	servName := generator.CamelCase(origServName)

	innerServerType := servName + "Server"
	handlerName := unexport(servName) + "Handler"

	g.P("type ", handlerName, " struct {")
	g.P("ss ", innerServerType)
	g.P("}")
	g.P()

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

		var (
			pattern    string
			httpMethod string
		)

		switch x := info.GetPattern().(type) {
		case *fhannotations.HttpRule_Get:
			httpMethod = "GET"
			pattern = x.Get
		case *fhannotations.HttpRule_Post:
			httpMethod = "POST"
			pattern = x.Post
		case *fhannotations.HttpRule_Put:
			httpMethod = "PUT"
			pattern = x.Put
		case *fhannotations.HttpRule_Patch:
			httpMethod = "PATCH"
			pattern = x.Patch
		case *fhannotations.HttpRule_Delete:
			httpMethod = "DELETE"
			pattern = x.Delete
		default:
			g.gen.Fail("xyz.featherhead.http requires a method: pattern")
		}

		vars, err := router.ExtractVariables(pattern)
		if err != nil {
			g.gen.Error(err)
			return
		}

		//g.gen.GetFieldName(message *generator.Descriptor, field *descriptor.FieldDescriptorProto)

		handlerMethod := g.generateServerCallName(servName, method)
		g.P("func (h* ", handlerName, " )", handlerMethod, "(ctx context_svchttp.Context, rw http_svchttp.ResponseWriter, req *http_svchttp.Request) error {")
		g.P("// info:      ", info.String())
		g.P("//   method:  ", httpMethod)
		g.P("//   pattern: ", pattern)
		g.P("//   input:   ", inputType.String())
		g.P("//   output:  ", outputType.String())
		g.P("if req.Method != ", strconv.Quote(httpMethod), "{")
		g.P(`  return errors_svchttp.MethodNotAllowedf("expected `, httpMethod, ` request")`)
		g.P("}")
		g.P()

		g.P(`var (`)
		g.P(`input `, g.typeName(inputTypeName))
		if len(vars) > 0 {
			g.P(`params = router_svchttp.P(ctx)`)
		}
		g.P(`)`)
		g.P()

		if httpMethod == "POST" || httpMethod == "PUT" || httpMethod == "PATCH" {
			g.P("{ // from body")
			g.P("err := json_svchttp.NewDecoder(req.Body).Decode(&input)")
			g.P(`if err != nil {`)
			g.P(`return errors_svchttp.Trace(err)`)
			g.P(`}`)
			g.P(`}`)
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
		} else {
			g.P(`err := h.ss.`, g.generateServerCall(servName, method))
		}
		g.P(`if err != nil {`)
		g.P(`return errors_svchttp.Trace(err)`)
		g.P(`}`)
		g.P(`_ = output`)
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
