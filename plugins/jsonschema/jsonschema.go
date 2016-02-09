package jsonschema

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/limbo-services/protobuf/proto"
	pb "github.com/limbo-services/protobuf/protoc-gen-gogo/descriptor"
	"github.com/limbo-services/protobuf/protoc-gen-gogo/generator"

	"github.com/limbo-services/core/runtime/limbo"
)

func init() {
	generator.RegisterPlugin(new(jsonschema))
}

// grpc is an implementation of the Go protocol buffer compiler's
// plugin architecture.  It generates bindings for gRPC support.
type jsonschema struct {
	gen          *generator.Generator
	definitions  map[string]interface{}
	dependencies map[string][]string
	imports      generator.PluginImports
	runtimePkg   generator.Single
	initCounter  int
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
	g.runtimePkg = imp.NewImport("github.com/limbo-services/core/runtime/limbo")
	g.definitions = map[string]interface{}{}
	g.dependencies = map[string][]string{}

	for i, message := range file.Messages() {
		g.generateMessageSchema(file, message, i)
	}

	for i, enum := range file.Enums() {
		g.generateEnumSchema(file, enum, i)
	}

	g.initCounter++
	defVarName := fmt.Sprintf("jsonSchemaDefs%d", g.initCounter)
	g.gen.AddInitf(`%s.RegisterSchemaDefinitions(%s)`, g.runtimePkg.Use(), defVarName)
	g.P(`var `+defVarName+` = []`, g.runtimePkg.Use(), `.SchemaDefinition{`)
	for name, def := range g.definitions {
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
		"id":    typeName,
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
		schema["id"] = typeName
	}

	{
		sort.Strings(dependencies)
		var last string
		tmp := dependencies[:0]
		for _, dep := range dependencies {
			if dep == last {
				continue
			}
			last = dep
			tmp = append(tmp, dep)
		}
		dependencies = tmp
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
		if f := limbo.GetFormat(field); f != "" {
			def["format"] = f
		}
		if p, ok := limbo.GetPattern(field); ok {
			def["pattern"] = p
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
