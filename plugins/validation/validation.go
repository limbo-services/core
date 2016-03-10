package plugin

import (
	"fmt"
	"strconv"

	"github.com/limbo-services/core/runtime/limbo"
	"github.com/limbo-services/protobuf/gogoproto"
	pb "github.com/limbo-services/protobuf/protoc-gen-gogo/descriptor"
	"github.com/limbo-services/protobuf/protoc-gen-gogo/generator"
)

func init() {
	generator.RegisterPlugin(new(validation))
}

type validation struct {
	gen *generator.Generator

	imports   generator.PluginImports
	errorsPkg generator.Single
	regexpPkg generator.Single
}

func (g *validation) Name() string {
	return "validation"
}

func (g *validation) Init(gen *generator.Generator) {
	g.gen = gen
}

func (g *validation) P(args ...interface{}) { g.gen.P(args...) }

func (g *validation) Generate(file *generator.FileDescriptor) {
	imp := generator.NewPluginImports(g.gen)
	g.imports = imp
	g.errorsPkg = imp.NewImport("github.com/juju/errors")
	g.regexpPkg = imp.NewImport("regexp")

	for _, msg := range file.Messages() {
		g.generateValidator(file, msg)
	}

}

func (g *validation) GenerateImports(file *generator.FileDescriptor) {
	g.imports.GenerateImports(file)
}

func (g *validation) generateValidator(file *generator.FileDescriptor, msg *generator.Descriptor) {
	if msg.GetOptions().GetMapEntry() {
		return
	}

	g.P(`func (msg *`, g.gen.TypeName(msg), `) Validate() error {`)

	var patterns = map[string]string{}

	for idx, field := range msg.Field {
		if field.OneofIndex != nil {
			continue
		}

		fieldName := "msg." + g.gen.GetFieldName(msg, field)
		g.generateTests(msg, field, fieldName, idx, patterns)
	}

	for oneofIdx, oneof := range msg.OneofDecl {
		g.P(`switch c := msg.Get`, generator.CamelCase(oneof.GetName()), `().(type) {`)
		for idx, field := range msg.Field {
			if field.OneofIndex == nil {
				continue
			}
			if *field.OneofIndex != int32(oneofIdx) {
				continue
			}

			g.P(`case *`, g.gen.OneOfTypeName(msg, field), `:`)
			fieldName := "c." + g.gen.GetOneOfFieldName(msg, field)
			g.generateTests(msg, field, fieldName, idx, patterns)
		}
		g.P(`}`)
	}

	g.P(`return nil`)
	g.P(`}`)
	g.P(``)

	for name, pattern := range patterns {
		g.P(`var `, name, ` = `, g.regexpPkg.Use(), `.MustCompile(`, strconv.Quote(pattern), `)`)
	}
	g.P(``)
}

func (g *validation) generateTests(msg *generator.Descriptor, field *pb.FieldDescriptorProto, fieldName string, idx int, patterns map[string]string) {

	if limbo.IsRequiredProperty(field) {
		g.generateRequiredTest(msg, field, fieldName)
	}

	if n, ok := limbo.GetMinItems(field); ok {
		g.generateMinItemsTest(msg, field, fieldName, int(n))
	}

	if n, ok := limbo.GetMaxItems(field); ok {
		g.generateMaxItemsTest(msg, field, fieldName, int(n))
	}

	if pattern, ok := limbo.GetPattern(field); ok {
		patternVar := fmt.Sprintf("valPattern_%s_%d", msg.GetName(), idx)
		patterns[patternVar] = pattern
		g.generatePatternTest(msg, field, fieldName, pattern, patternVar)
	}

	if n, ok := limbo.GetMinLength(field); ok {
		g.generateMinLengthTest(msg, field, fieldName, int(n))
	}

	if n, ok := limbo.GetMaxLength(field); ok {
		g.generateMaxLengthTest(msg, field, fieldName, int(n))
	}

	if field.GetType() == pb.FieldDescriptorProto_TYPE_MESSAGE {
		g.generateSubMessageTest(msg, field, fieldName)
	}

}

func (g *validation) generateRequiredTest(msg *generator.Descriptor, field *pb.FieldDescriptorProto, fieldName string) {

	if field.IsRepeated() {
		g.P(`if `, fieldName, ` == nil || len(`, fieldName, `) == 0 {`)
		g.P(`return `, g.errorsPkg.Use(), `.NotValidf("`, field.Name, ` is required")`)
		g.P(`}`)
		return
	}

	switch field.GetType() {
	case pb.FieldDescriptorProto_TYPE_BYTES:
		g.P(`if `, fieldName, ` == nil || len(`, fieldName, `) == 0 {`)
		g.P(`return `, g.errorsPkg.Use(), `.NotValidf("`, field.Name, ` is required")`)
		g.P(`}`)
	case pb.FieldDescriptorProto_TYPE_STRING:
		g.P(`if `, fieldName, ` == "" {`)
		g.P(`return `, g.errorsPkg.Use(), `.NotValidf("`, field.Name, ` is required")`)
		g.P(`}`)
	case pb.FieldDescriptorProto_TYPE_MESSAGE:
		if gogoproto.IsNullable(field) {
			g.P(`if `, fieldName, ` == nil {`)
			g.P(`return `, g.errorsPkg.Use(), `.NotValidf("`, field.Name, ` is required")`)
			g.P(`}`)
		}
	}

}

func (g *validation) generateSubMessageTest(msg *generator.Descriptor, field *pb.FieldDescriptorProto, fieldName string) {

	var casttyp = "value"
	var typ = g.gen.TypeName(g.gen.ObjectNamed(field.GetTypeName()))
	var zeroValuer = "value"
	if gogoproto.IsCastType(field) {
		if gogoproto.IsNullable(field) {
			casttyp = "((*" + typ + ")(" + casttyp + "))"
		} else {
			casttyp = "((*" + typ + ")(&" + casttyp + "))"
			zeroValuer = "((" + typ + ")(value))"
		}
	}

	if g.gen.IsMap(field) {
		// g.gen.GetMapKeyField(field*descriptor.FieldDescriptorProto, keyField*descriptor.FieldDescriptorProto)

	} else if field.IsRepeated() {
		g.P(`for _, value :=range `, fieldName, `{`)
		if gogoproto.IsNullable(field) {
			g.P(`if value != nil {`)
			g.P(`if err := `, casttyp, `.Validate(); err != nil {`)
			g.P(`return `, g.errorsPkg.Use(), `.Trace(err)`)
			g.P(`}`)
			g.P(`}`)
		} else {
			g.P(`if (`, typ, `{}) != `, zeroValuer, ` {`)
			g.P(`if err := `, casttyp, `.Validate(); err != nil {`)
			g.P(`return `, g.errorsPkg.Use(), `.Trace(err)`)
			g.P(`}`)
			g.P(`}`)
		}
		g.P(`}`)
	} else {
		if gogoproto.IsNullable(field) {
			g.P(`{`)
			g.P(`value := `, fieldName)
			g.P(`if value != nil {`)
			g.P(`if err := `, casttyp, `.Validate(); err != nil {`)
			g.P(`return `, g.errorsPkg.Use(), `.Trace(err)`)
			g.P(`}`)
			g.P(`}`)
			g.P(`}`)
		} else {
			g.P(`{`)
			g.P(`value := `, fieldName)
			g.P(`if (`, typ, `{}) != `, zeroValuer, ` {`)
			g.P(`if err := `, casttyp, `.Validate(); err != nil {`)
			g.P(`return `, g.errorsPkg.Use(), `.Trace(err)`)
			g.P(`}`)
			g.P(`}`)
			g.P(`}`)
		}
	}

}

func (g *validation) generateMinItemsTest(msg *generator.Descriptor, field *pb.FieldDescriptorProto, fieldName string, minItems int) {

	if !field.IsRepeated() {
		g.gen.Fail("limbo.minItems can only be used on repeated fields.")
	}

	g.P(`if len(`, fieldName, `) < `, minItems, ` {`)
	g.P(`return `, g.errorsPkg.Use(), `.NotValidf("number of items in `, field.Name, ` (%q) is to small (minimum: `, minItems, `)", len(`, fieldName, `))`)
	g.P(`}`)
}

func (g *validation) generateMaxItemsTest(msg *generator.Descriptor, field *pb.FieldDescriptorProto, fieldName string, maxItems int) {

	if !field.IsRepeated() {
		g.gen.Fail("limbo.maxItems can only be used on repeated fields.")
	}

	g.P(`if len(`, fieldName, `) < `, maxItems, ` {`)
	g.P(`return `, g.errorsPkg.Use(), `.NotValidf("number of items in `, field.Name, ` (%q) is to small (maximum: `, maxItems, `)", len(`, fieldName, `))`)
	g.P(`}`)
}

func (g *validation) generateMinLengthTest(msg *generator.Descriptor, field *pb.FieldDescriptorProto, fieldName string, minLength int) {

	if field.GetType() != pb.FieldDescriptorProto_TYPE_STRING {
		g.gen.Fail("limbo.minLength can only be used on string fields.")
	}

	if field.IsRepeated() {
		g.P(`for idx, value :=range `, fieldName, `{`)
		g.P(`if len(value) < `, minLength, ` {`)
		g.P(`return `, g.errorsPkg.Use(), `.NotValidf("`, field.Name, `[%d] (%q) is to short (minimum length: `, minLength, `)", idx, value)`)
		g.P(`}`)
		g.P(`}`)
		return
	}

	g.P(`if len(`, fieldName, `) < `, minLength, ` {`)
	g.P(`return `, g.errorsPkg.Use(), `.NotValidf("`, field.Name, ` (%q) is to short (minimum length: `, minLength, `)", `, fieldName, `)`)
	g.P(`}`)

}

func (g *validation) generateMaxLengthTest(msg *generator.Descriptor, field *pb.FieldDescriptorProto, fieldName string, maxLength int) {

	if field.GetType() != pb.FieldDescriptorProto_TYPE_STRING {
		g.gen.Fail("limbo.maxLength can only be used on string fields.")
	}

	if field.IsRepeated() {
		g.P(`for idx, value :=range `, fieldName, `{`)
		g.P(`if len(value) > `, maxLength, ` {`)
		g.P(`return `, g.errorsPkg.Use(), `.NotValidf("`, field.Name, `[%d] (%q) is to long (maximum length: `, maxLength, `)", idx, value)`)
		g.P(`}`)
		g.P(`}`)
		return
	}

	g.P(`if len(`, fieldName, `) > `, maxLength, ` {`)
	g.P(`return `, g.errorsPkg.Use(), `.NotValidf("`, field.Name, ` (%q) is to long (maximum length: `, maxLength, `)", `, fieldName, `)`)
	g.P(`}`)

}

func (g *validation) generatePatternTest(msg *generator.Descriptor, field *pb.FieldDescriptorProto, fieldName string, pattern, patternVar string) {

	if field.GetType() != pb.FieldDescriptorProto_TYPE_STRING {
		g.gen.Fail("limbo.pattern can only be used on string fields.")
	}

	if field.IsRepeated() {
		g.P(`for idx, value :=range `, fieldName, `{`)
		g.P(`if !`, patternVar, `.MatchString(value) {`)
		g.P(`return `, g.errorsPkg.Use(), `.NotValidf("`, field.Name, `[%d] (%q) does not match pattern %s", idx, value, `, strconv.Quote(pattern), `)`)
		g.P(`}`)
		g.P(`}`)
		return
	}

	g.P(`if !`, patternVar, `.MatchString(`, fieldName, `) {`)
	g.P(`return `, g.errorsPkg.Use(), `.NotValidf("`, field.Name, ` (%q) does not match pattern %s", `, fieldName, `, `, strconv.Quote(pattern), `)`)
	g.P(`}`)

}
