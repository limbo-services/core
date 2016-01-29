package svchttp

import (
	"encoding/json"
	"path"
	"strings"

	"github.com/gogo/protobuf/proto"
	pb "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
	plugin "github.com/gogo/protobuf/protoc-gen-gogo/plugin"

	"github.com/fd/featherhead/pkg/api/httpapi/router"
)

type SwaggerInfo struct {
	Title          string `json:"title"`
	Description    string `json:"description,omitempty"`
	TermsOfService string `json:"termsOfService,omitempty"`
	Version        string `json:"version"`
}

type SwaggerParameterObject struct {
	Name     string      `json:"name,omitempty"`
	In       string      `json:"in"`
	Type     string      `json:"type,omitempty"`
	Pattern  string      `json:"pattern,omitempty"`
	Required bool        `json:"required,omitempty"`
	MinItems int         `json:"minItems,omitempty"`
	MaxItems int         `json:"maxItems,omitempty"`
	Schema   interface{} `json:"schema,omitempty"`
}

type SwaggerResponseObject struct {
	Description string      `json:"description"`
	Schema      interface{} `json:"schema"`
}

type SwaggerOperationObject struct {
	Summary     string                           `json:"summary,omitempty"`
	Description string                           `json:"description,omitempty"`
	Parameters  []SwaggerParameterObject         `json:"parameters,omitempty"`
	Responses   map[string]SwaggerResponseObject `json:"responses"`
	Tags        []string                         `json:"tags"`
}

type SwaggerRoot struct {
	Swagger string      `json:"swagger"`
	Info    SwaggerInfo `json:"info"`
	// host
	// basePath
	Schemes  []string `json:"schemes"`  // ["https", "wss"]
	Consumes []string `json:"consumes"` // ["application/json; charset=utf-8"]
	Produces []string `json:"produces"` // ["application/json; charset=utf-8"]

	// swagger.Paths["<path>"]
	Paths map[string]map[string]*SwaggerOperationObject `json:"paths"`

	Definitions map[string]interface{} `json:"definitions"`
}

func (g *svchttp) loadSwaggerSpec(file *generator.FileDescriptor) *SwaggerRoot {
	expected := swaggerSpecFileName(file.GetName())

	for _, file := range g.gen.Response.File {
		if file.GetName() != expected {
			continue
		}

		var root *SwaggerRoot
		err := json.Unmarshal([]byte(file.GetContent()), &root)
		if err != nil {
			g.gen.Error(err)
		}

		return root
	}

	return &SwaggerRoot{
		Swagger: "2.0",
		Info: SwaggerInfo{
			Title:   g.gen.PackageImportPath,
			Version: "1",
		},

		Schemes:  []string{"https", "wss"},
		Consumes: []string{"application/json; charset=utf-8"},
		Produces: []string{"application/json; charset=utf-8"},

		Paths: map[string]map[string]*SwaggerOperationObject{},
	}
}

func (g *svchttp) generateSwaggerSpec(file *generator.FileDescriptor, service *pb.ServiceDescriptorProto, apis []*API) {
	spec := g.loadSwaggerSpec(file)
	defer g.saveSwaggerSpec(file, spec)

	for _, api := range apis {
		method, pattern, ok := api.GetMethodAndPattern()
		if !ok {
			continue
		}

		pattern = strings.Replace(pattern, ".", "_", -1)

		query := ""
		if idx := strings.IndexByte(pattern, '?'); idx >= 0 {
			query = pattern[idx+1:]
			pattern = pattern[:idx]
		}
		_ = query

		pathItem := spec.Paths[pattern]
		if pathItem == nil {
			pathItem = make(map[string]*SwaggerOperationObject)
			spec.Paths[pattern] = pathItem
		}

		operation := pathItem[strings.ToLower(method)]
		if operation == nil {
			operation = &SwaggerOperationObject{}
			pathItem[strings.ToLower(method)] = operation
		}

		if operation.Responses == nil {
			operation.Responses = make(map[string]SwaggerResponseObject)
		}

		operation.Tags = append(operation.Tags, api.service.GetName())

		// inputType := g.gen.ObjectNamed(api.method.GetInputType()).(*generator.Descriptor)
		outputType := g.gen.ObjectNamed(api.method.GetOutputType()).(*generator.Descriptor)

		operation.Description = strings.TrimSpace(g.gen.Comments(api.descIndexPath))
		operation.Responses["200"] = SwaggerResponseObject{
			Description: "Response on success",
			Schema: map[string]string{
				"$ref": "#/definitions/" + file.GetPackage() + "." + strings.Join(outputType.TypeName(), "."),
			},
		}

		g.appendPathParameters(operation, pattern)
		g.appendQueryParameters(operation, query)

		if method != "HEAD" && method != "GET" && method != "DELETE" {
			g.appendBodyParameter(operation, api.method.GetInputType())
		}
	}

}

func (g *svchttp) appendPathParameters(operation *SwaggerOperationObject, pattern string) {

	vars, err := router.ExtractVariables(pattern)
	if err != nil {
		g.gen.Error(err)
		return
	}

	for _, v := range vars {
		p := SwaggerParameterObject{
			Name: v.Name,
			In:   "path",
			Type: "string",
			//Format:
		}

		if v.Pattern != "" {
			p.Pattern = v.Pattern
		}

		if v.MinCount > 0 {
			p.Required = true
			p.MinItems = v.MinCount
		}

		if v.MaxCount > 0 {
			p.MaxItems = v.MaxCount
		}

		operation.Parameters = append(operation.Parameters, p)

	}
}

func (g *svchttp) appendQueryParameters(operation *SwaggerOperationObject, queryString string) {
	if queryString == "" {
		return
	}

	queryParams := map[string]string{}
	for _, pair := range strings.SplitN(queryString, "&", -1) {
		idx := strings.Index(pair, "={")
		if pair[len(pair)-1] != '}' || idx < 0 {
			g.gen.Fail("invalid query paramter")
		}
		queryParams[pair[:idx]] = pair[idx+2 : len(pair)-1]
	}

	for k, v := range queryParams {
		_ = v

		p := SwaggerParameterObject{
			Name: k,
			In:   "query",
			Type: "string",
			//Format:
		}

		operation.Parameters = append(operation.Parameters, p)

	}
}

func (g *svchttp) appendBodyParameter(operation *SwaggerOperationObject, inputType string) {

	p := SwaggerParameterObject{
		Name: "parameters",
		In:   "body",
		Schema: map[string]interface{}{
			"$ref": "#/definitions/" + strings.TrimPrefix(inputType, "."),
		},
	}

	operation.Parameters = append(operation.Parameters, p)

}

func (g *svchttp) saveSwaggerSpec(file *generator.FileDescriptor, root *SwaggerRoot) {
	if strings.HasPrefix(file.GetName(), "google/") {
		return
	}

	data, err := json.Marshal(&root)
	if err != nil {
		g.gen.Error(err)
	}

	expected := swaggerSpecFileName(file.GetName())
	for _, file := range g.gen.Response.File {
		if file.GetName() != expected {
			continue
		}

		file.Content = proto.String(string(data))
		return
	}

	g.gen.Response.File = append(g.gen.Response.File, &plugin.CodeGeneratorResponse_File{
		Name:    proto.String(expected),
		Content: proto.String(string(data)),
	})
}

func swaggerSpecFileName(name string) string {
	return path.Dir(name) + "/swagger.json"
}
