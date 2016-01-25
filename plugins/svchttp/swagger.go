package svchttp

import (
	"encoding/json"
	"path"
	"strings"

	"github.com/fd/featherhead/pkg/api/httpapi/router"
	"github.com/gogo/protobuf/proto"
	pb "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
	plugin "github.com/gogo/protobuf/protoc-gen-gogo/plugin"
)

type SwaggerInfo struct {
	Title          string `json:"title"`
	Description    string `json:"description,omitempty"`
	TermsOfService string `json:"termsOfService,omitempty"`
	Version        string `json:"version"`
}

type SwaggerParameterObject struct {
	Name     string `json:"name"`
	In       string `json:"in"`
	Type     string `json:"type"`
	Pattern  string `json:"pattern,omitempty"`
	Required bool   `json:"required,omitempty"`
	MinItems int    `json:"minItems,omitempty"`
	MaxItems int    `json:"maxItems,omitempty"`
}

type SwaggerResponseObject struct {
	Description string      `json:"description"`
	Schema      interface{} `json:"schema"`
	// schema      string
	// headers
	// examples
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
}

func (g *svchttp) loadSwaggerSpec() *SwaggerRoot {
	for _, file := range g.gen.Response.File {
		if path.Base(file.GetName()) != "swagger.json" {
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
	spec := g.loadSwaggerSpec()
	defer g.saveSwaggerSpec(file, spec)

	for _, api := range apis {
		method, pattern, ok := api.GetMethodAndPattern()
		if !ok {
			continue
		}

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

		outputType := g.gen.ObjectNamed(api.method.GetOutputType()).(*generator.Descriptor)

		operation.Description = strings.TrimSpace(g.gen.Comments(api.descIndexPath))
		operation.Responses["200"] = SwaggerResponseObject{
			Description: "Response on success",
			Schema:      messageToSchema(g.gen, outputType),
		}

		g.appendPathParameters(operation, pattern)
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

func (g *svchttp) saveSwaggerSpec(file *generator.FileDescriptor, root *SwaggerRoot) {
	data, err := json.Marshal(&root)
	if err != nil {
		g.gen.Error(err)
	}

	for _, file := range g.gen.Response.File {
		if path.Base(file.GetName()) != "swagger.json" {
			continue
		}

		file.Content = proto.String(string(data))
		return
	}

	g.gen.Response.File = append(g.gen.Response.File, &plugin.CodeGeneratorResponse_File{
		Name:    proto.String(swaggerSpecFileName(file.GetName())),
		Content: proto.String(string(data)),
	})
}

func swaggerSpecFileName(name string) string {
	return path.Dir(name) + "/swagger.json"
}
