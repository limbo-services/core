package jsonschema

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"limbo.services/protobuf/proto"
	pb "limbo.services/protobuf/protoc-gen-gogo/descriptor"
	"limbo.services/protobuf/protoc-gen-gogo/generator"

	"limbo.services/core/runtime/limbo"
	"limbo.services/core/runtime/router"
)

func init() {
	generator.RegisterPlugin(new(jsonschema))
}

// grpc is an implementation of the Go protocol buffer compiler's
// plugin architecture.  It generates bindings for gRPC support.
type jsonschema struct {
	gen          *generator.Generator
	operations   []*operationDecl
	definitions  map[string]interface{}
	dependencies map[string][]string
	imports      generator.PluginImports
	runtimePkg   generator.Single
}

type operationDecl struct {
	Dependencies []string
	Pattern      string
	Method       string
	Swagger      interface{}
}

// Name returns the name of this plugin, "jsonschema".
func (g *jsonschema) Name() string {
	return "jsonschema"
}

// reservedClientName records whether a client name is reserved on the client side.
var reservedClientName = map[string]bool{
// TODO: do we need any in gRPC?
}

// Init initializes the plugin.
func (g *jsonschema) Init(gen *generator.Generator) {
	g.gen = gen
}

// Given a type name defined in a .proto, return its object.
// Also record that we're using it, to guarantee the associated import.
func (g *jsonschema) objectNamed(name string) generator.Object {
	g.gen.RecordTypeUse(name)
	return g.gen.ObjectNamed(name)
}

// Given a type name defined in a .proto, return its name as we will print it.
func (g *jsonschema) typeName(str string) string {
	return g.gen.TypeName(g.objectNamed(str))
}

// P forwards to g.gen.P.
func (g *jsonschema) P(args ...interface{}) { g.gen.P(args...) }

// Generate generates code for the services in the given file.
func (g *jsonschema) Generate(file *generator.FileDescriptor) {
	imp := generator.NewPluginImports(g.gen)
	g.imports = imp
	g.runtimePkg = imp.NewImport("limbo.services/core/runtime/limbo")
	g.operations = nil
	g.definitions = map[string]interface{}{}
	g.dependencies = map[string][]string{}

	for i, enum := range file.Enums() {
		g.generateEnumSchema(file, enum, i)
	}

	for i, message := range file.Messages() {
		g.generateMessageSchema(file, message, i)
	}

	for i, service := range file.Service {
		g.generateServiceSchema(file, service, i)
	}

	if len(g.definitions) > 0 {
		defVarName := fmt.Sprintf("jsonSchemaDefs%x", sha1.Sum([]byte(file.GetName())))
		g.gen.AddInitf(`%s.RegisterSchemaDefinitions(%s)`, g.runtimePkg.Use(), defVarName)
		g.P(`var `+defVarName+` = []`, g.runtimePkg.Use(), `.SchemaDefinition{`)
		var definitionNames = make([]string, 0, len(g.definitions))
		for name := range g.definitions {
			definitionNames = append(definitionNames, name)
		}
		sort.Strings(definitionNames)
		for _, name := range definitionNames {
			def := g.definitions[name]

			data, err := json.MarshalIndent(def, "\t\t", "\t")
			if err != nil {
				panic(err)
			}
			dataStr := strings.Replace(string(data), "`", "`+\"`\"+`", -1)

			g.P(`{`)
			g.P(`Name: `, strconv.Quote(name), `,`)
			g.P(`Dependencies: []string{`)
			for _, dep := range g.dependencies[name] {
				g.P(strconv.Quote(dep), ",")
			}
			g.P(`},`)
			g.P(`Definition: []byte(`, "`", dataStr, "`),")
			g.P(`},`)
		}
		g.P(`}`)
	}

	if len(g.operations) > 0 {
		defVarName := fmt.Sprintf("swaggerDefs%x", sha1.Sum([]byte(file.GetName())))
		g.gen.AddInitf(`%s.RegisterSwaggerOperations(%s)`, g.runtimePkg.Use(), defVarName)
		g.P(`var `+defVarName+` = []`, g.runtimePkg.Use(), `.SwaggerOperation{`)
		for _, op := range g.operations {

			data, err := json.MarshalIndent(op.Swagger, "\t\t", "\t")
			if err != nil {
				panic(err)
			}
			dataStr := strings.Replace(string(data), "`", "`+\"`\"+`", -1)

			g.P(`{`)
			g.P(`Pattern: `, strconv.Quote(op.Pattern), `,`)
			g.P(`Method: `, strconv.Quote(op.Method), `,`)
			g.P(`Dependencies: []string{`)
			for _, dep := range op.Dependencies {
				g.P(strconv.Quote(dep), ",")
			}
			g.P(`},`)
			g.P(`Definition: []byte(`, "`", dataStr, "`),")
			g.P(`},`)
		}
		g.P(`}`)
	}
}

// GenerateImports generates the import declaration for this file.
func (g *jsonschema) GenerateImports(file *generator.FileDescriptor) {
	g.imports.GenerateImports(file)
}

func unexport(s string) string { return strings.ToLower(s[:1]) + s[1:] }

func (g *jsonschema) generateMessageSchema(file *generator.FileDescriptor, msg *generator.Descriptor, index int) {
	typeName := file.GetPackage() + "." + strings.Join(msg.TypeName(), ".")
	typeName = strings.TrimPrefix(typeName, ".")

	if g.definitions[typeName] == nil {
		def, deps := g.messageToSchema(file, msg)
		g.definitions[typeName] = def
		g.dependencies[typeName] = deps
	}
}

func (g *jsonschema) generateServiceSchema(file *generator.FileDescriptor, srv *pb.ServiceDescriptorProto, index int) {
	path := fmt.Sprintf("6,%d", index)

	for idx, method := range srv.GetMethod() {
		rule := limbo.GetHTTPRule(method)
		if rule == nil {
			continue
		}

		comment := g.gen.Comments(fmt.Sprintf("%s,%d,%d", path, 2, idx))
		comment = strings.TrimSpace(comment)

		g.generateServiceMethodSchema(file, srv, method, rule, comment)
	}

}

func (g *jsonschema) generateServiceMethodSchema(file *generator.FileDescriptor, srv *pb.ServiceDescriptorProto, meth *pb.MethodDescriptorProto, rule *limbo.HttpRule, comment string) {
	var (
		method  string
		pattern string
	)

	switch p := rule.GetPattern().(type) {
	case *limbo.HttpRule_Delete:
		method = "DELETE"
		pattern = p.Delete
	case *limbo.HttpRule_Get:
		method = "GET"
		pattern = p.Get
	case *limbo.HttpRule_Post:
		method = "POST"
		pattern = p.Post
	case *limbo.HttpRule_Patch:
		method = "PATCH"
		pattern = p.Patch
	case *limbo.HttpRule_Put:
		method = "PUT"
		pattern = p.Put
	default:
		panic("unknown pattern type")
	}

	query := ""
	if idx := strings.IndexByte(pattern, '?'); idx >= 0 {
		query = pattern[idx+1:]
		pattern = pattern[:idx]
	}

	path := regexp.MustCompile("\\{.+\\}").ReplaceAllStringFunc(pattern, func(v string) string {
		return strings.Replace(v, ".", "_", -1)
	})

	input := strings.TrimPrefix(meth.GetInputType(), ".")
	output := strings.TrimPrefix(meth.GetOutputType(), ".")

	var outputSchema interface{} = map[string]string{"$ref": output}
	if meth.GetServerStreaming() {
		outputSchema = map[string]interface{}{
			"type":  "array",
			"items": outputSchema,
		}
	}

	var tags = []string{srv.GetName()}
	tags = append(tags, rule.Tags...)

	op := map[string]interface{}{
		"tags":        tags,
		"description": comment,
		"responses": map[string]interface{}{
			"200": map[string]interface{}{
				"description": "",
				"schema":      outputSchema,
			},
		},
	}

	var parameters []map[string]interface{}

	if params := g.collectPathParameters(pattern); len(params) > 0 {
		parameters = append(parameters, params...)
	}

	if params := g.collectQueryParameters(query); len(params) > 0 {
		parameters = append(parameters, params...)
	}

	if method != "HEAD" && method != "GET" && method != "OPTIONS" && method != "DELETE" {
		parameters = append(parameters, map[string]interface{}{
			"name": "parameters",
			"in":   "body",
			"schema": map[string]interface{}{
				"$ref": input,
			},
		})
	}

	if len(parameters) > 0 {
		op["parameters"] = parameters
	}

	if scope, ok := limbo.GetScope(meth); ok {
		op["security"] = []map[string][]string{
			{"oauth": {scope}},
		}
	}

	decl := &operationDecl{
		Pattern:      path,
		Method:       method,
		Dependencies: uniqStrings([]string{input, output}),
		Swagger:      op,
	}
	g.operations = append(g.operations, decl)

	// x, _ := json.MarshalIndent(op, "  ", "  ")
	// fmt.Fprintf(os.Stderr, "%s %s:\n  %s\n", method, path, x)

	for _, a := range rule.GetAlternatives() {
		g.generateServiceMethodSchema(file, srv, meth, a, comment)
	}
}

func (g *jsonschema) collectPathParameters(pattern string) []map[string]interface{} {
	vars, err := router.ExtractVariables(pattern)
	if err != nil {
		g.gen.Error(err)
		return nil
	}

	params := make([]map[string]interface{}, 0, len(vars))

	for _, v := range vars {
		param := map[string]interface{}{
			"name": strings.Replace(v.Name, ".", "_", -1),
			"in":   "path",
			"type": "string",
			//"format":
		}

		if v.Pattern != "" {
			param["pattern"] = v.Pattern
		}

		if v.MinCount > 0 {
			param["required"] = true
			param["minItems"] = v.MinCount
		}

		if v.MaxCount > 0 {
			param["maxItems"] = v.MaxCount
		}

		params = append(params, param)
	}

	return params
}

func (g *jsonschema) collectQueryParameters(queryString string) []map[string]interface{} {
	if queryString == "" {
		return nil
	}

	queryParams := map[string]string{}
	for _, pair := range strings.SplitN(queryString, "&", -1) {
		idx := strings.Index(pair, "={")
		if pair[len(pair)-1] != '}' || idx < 0 {
			g.gen.Fail("invalid query paramter")
		}
		queryParams[pair[:idx]] = pair[idx+2 : len(pair)-1]
	}

	params := make([]map[string]interface{}, 0, len(queryParams))

	for k, v := range queryParams {
		_ = v

		p := map[string]interface{}{
			"name": k,
			"in":   "query",
			"type": "string",
			//"format":
		}

		params = append(params, p)
	}

	return params
}

func (g *jsonschema) generateEnumSchema(file *generator.FileDescriptor, enum *generator.EnumDescriptor, index int) {
	typeName := file.GetPackage() + "." + strings.Join(enum.TypeName(), ".")
	typeName = strings.TrimPrefix(typeName, ".")

	def := g.definitions[typeName]
	if def == nil {
		def = g.enumToSchema(file, enum)
		g.definitions[typeName] = def
	}
}

func (g *jsonschema) enumToSchema(file *generator.FileDescriptor, desc *generator.EnumDescriptor) interface{} {
	typeName := file.GetPackage() + "." + strings.Join(desc.TypeName(), ".")
	typeName = strings.TrimPrefix(typeName, ".")

	title := desc.TypeName()[len(desc.TypeName())-1]

	values := make([]interface{}, 0, 2*len(desc.Value))
	for _, x := range desc.Value {
		values = append(values, x.GetNumber())
		values = append(values, x.GetName())
	}

	return map[string]interface{}{
		// "id":    typeName,
		"enum":  values,
		"title": title,
	}
}

func (g *jsonschema) messageToSchema(file *generator.FileDescriptor, desc *generator.Descriptor) (interface{}, []string) {
	typeName := file.GetPackage() + "." + strings.Join(desc.TypeName(), ".")
	typeName = strings.TrimPrefix(typeName, ".")

	title := desc.TypeName()[len(desc.TypeName())-1]

	var (
		dependencies       []string
		properties         = make(map[string]interface{})
		requiredProperties []string
	)

	for i, field := range desc.GetField() {
		if field.OneofIndex != nil {
			continue
		}

		f, dep := g.fieldToSchema(field)
		if f == nil {
			continue
		}

		if limbo.IsRequiredProperty(field) {
			requiredProperties = append(requiredProperties, getJSONName(field))
		}

		{
			comment := g.gen.Comments(fmt.Sprintf("%s,%d,%d", desc.Path(), 2, i))
			comment = strings.TrimSpace(comment)
			if comment != "" {
				f["description"] = comment
			}
		}

		properties[getJSONName(field)] = f
		if dep != "" {
			dependencies = append(dependencies, dep)
		}
	}

	schema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}
	if len(requiredProperties) > 0 {
		schema["required"] = requiredProperties
	}

	if len(desc.OneofDecl) > 0 {
		allOffDefs := make([]interface{}, 0, 1+len(desc.OneofDecl))
		oneOfDefs := make([][]interface{}, len(desc.OneofDecl))
		for i, field := range desc.GetField() {
			if field.OneofIndex == nil {
				continue
			}
			oneofIndex := *field.OneofIndex

			f, dep := g.fieldToSchema(field)
			if f == nil {
				continue
			}

			if field.IsRepeated() {
				f = map[string]interface{}{
					"type":  "array",
					"items": f,
				}
			}

			{
				comment := g.gen.Comments(fmt.Sprintf("%s,%d,%d", desc.Path(), 2, i))
				comment = strings.TrimSpace(comment)
				if comment != "" {
					f["description"] = comment
				}
			}

			def := map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					getJSONName(field): f,
				},
				"required": []string{getJSONName(field)},
			}

			oneOfDefs[oneofIndex] = append(oneOfDefs[oneofIndex], def)
			if dep != "" {
				dependencies = append(dependencies, dep)
			}
		}

		allOffDefs = append(allOffDefs, schema)
		for i, defs := range oneOfDefs {
			def := map[string]interface{}{
				"oneOf": defs,
			}

			{
				comment := g.gen.Comments(fmt.Sprintf("%s,%d,%d", desc.Path(), 8, i))
				comment = strings.TrimSpace(comment)
				if comment != "" {
					def["description"] = comment
				}
			}

			allOffDefs = append(allOffDefs, def)
		}

		schema = map[string]interface{}{
			"type":  "object",
			"allOf": allOffDefs,
		}
	}

	{
		comment := g.gen.Comments(desc.Path())
		comment = strings.TrimSpace(comment)
		if comment != "" {
			schema["description"] = comment
		}
	}

	{
		schema["title"] = title
		// schema["id"] = typeName
	}

	{
		dependencies = uniqStrings(dependencies)
	}

	return schema, dependencies
}

func (g *jsonschema) fieldToSchema(field *pb.FieldDescriptorProto) (map[string]interface{}, string) {
	if field.Options != nil {
		v, _ := proto.GetExtension(field.Options, limbo.E_HideInSwagger)
		hidePtr, _ := v.(*bool)
		if hidePtr != nil && *hidePtr == true {
			return nil, ""
		}
	}

	var (
		def map[string]interface{}
		dep string
	)

	switch field.GetType() {

	case pb.FieldDescriptorProto_TYPE_BOOL:
		def = map[string]interface{}{
			"type": "boolean",
		}

	case pb.FieldDescriptorProto_TYPE_FLOAT:
		def = map[string]interface{}{
			"type":   "number",
			"format": "float",
		}

	case pb.FieldDescriptorProto_TYPE_DOUBLE:
		def = map[string]interface{}{
			"type":   "number",
			"format": "double",
		}

	case pb.FieldDescriptorProto_TYPE_FIXED32,
		pb.FieldDescriptorProto_TYPE_FIXED64,
		pb.FieldDescriptorProto_TYPE_UINT32,
		pb.FieldDescriptorProto_TYPE_UINT64:
		def = map[string]interface{}{
			"type": "integer",
		}

	case pb.FieldDescriptorProto_TYPE_INT32,
		pb.FieldDescriptorProto_TYPE_SFIXED32,
		pb.FieldDescriptorProto_TYPE_SINT32:
		def = map[string]interface{}{
			"type":   "integer",
			"format": "int32",
		}

	case pb.FieldDescriptorProto_TYPE_INT64,
		pb.FieldDescriptorProto_TYPE_SFIXED64,
		pb.FieldDescriptorProto_TYPE_SINT64:
		def = map[string]interface{}{
			"type":   "integer",
			"format": "int64",
		}

	case pb.FieldDescriptorProto_TYPE_STRING:
		def = map[string]interface{}{
			"type": "string",
		}
		if x, ok := limbo.GetFormat(field); ok {
			def["format"] = x
		}
		if x, ok := limbo.GetPattern(field); ok {
			def["pattern"] = x
		}
		if x, ok := limbo.GetMinLength(field); ok {
			def["minLength"] = x
		}
		if x, ok := limbo.GetMaxLength(field); ok {
			def["maxLength"] = x
		}

	case pb.FieldDescriptorProto_TYPE_BYTES:
		def = map[string]interface{}{
			"type":   "string",
			"format": "base64",
		}

	case pb.FieldDescriptorProto_TYPE_ENUM:
		dep = strings.TrimPrefix(field.GetTypeName(), ".")
		def = map[string]interface{}{
			"$ref": dep,
		}

	case pb.FieldDescriptorProto_TYPE_MESSAGE:
		dep = strings.TrimPrefix(field.GetTypeName(), ".")
		def = map[string]interface{}{
			"$ref": dep,
		}

	default:
		panic("unsupported " + field.GetType().String())

	}

	if field.IsRepeated() {
		def = map[string]interface{}{
			"type":  "array",
			"items": def,
		}
		if x, ok := limbo.GetMinItems(field); ok {
			def["minItems"] = x
		}
		if x, ok := limbo.GetMaxItems(field); ok {
			def["maxItems"] = x
		}
	}

	return def, dep
}

func getJSONName(field *pb.FieldDescriptorProto) string {
	name := field.GetJsonName()
	if name == "" {
		name = field.GetName()
	}
	return name
}

func uniqStrings(s []string) []string {
	sort.Strings(s)
	var last string
	var out = s[:0]
	for i, v := range s {
		if i == 0 || v != last {
			last = v
			out = append(out, v)
		}
	}
	return out
}
