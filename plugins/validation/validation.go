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
	g.P(`func (msg *`, msg.Name, `) Validate() error {`)

	var patterns = map[string]string{}

	for idx, field := range msg.Field {

		if limbo.IsRequiredProperty(field) {
			g.generateRequiredTest(msg, field)
		}

		if n, ok := limbo.GetMinItems(field); ok {
			g.generateMinItemsTest(msg, field, int(n))
		}

		if n, ok := limbo.GetMaxItems(field); ok {
			g.generateMaxItemsTest(msg, field, int(n))
		}

		if pattern, ok := limbo.GetPattern(field); ok {
			patternVar := fmt.Sprintf("valPattern_%s_%d", msg.GetName(), idx)
			patterns[patternVar] = pattern
			g.generatePatternTest(msg, field, pattern, patternVar)
		}

		if n, ok := limbo.GetMinLength(field); ok {
			g.generateMinLengthTest(msg, field, int(n))
		}

		if n, ok := limbo.GetMaxLength(field); ok {
			g.generateMaxLengthTest(msg, field, int(n))
		}

		if field.GetType() == pb.FieldDescriptorProto_TYPE_MESSAGE {
			g.generateSubMessageTest(msg, field)
		}

	}

	g.P(`return nil`)
	g.P(`}`)
	g.P(``)

	for name, pattern := range patterns {
		g.P(`var `, name, ` = `, g.regexpPkg.Use(), `.MustCompile(`, strconv.Quote(pattern), `)`)
	}
	g.P(``)
}

func (g *validation) generateRequiredTest(msg *generator.Descriptor, field *pb.FieldDescriptorProto) {
	fieldName := g.gen.GetFieldName(msg, field)

	if field.IsRepeated() {
		g.P(`if msg.`, fieldName, ` == nil || len(msg.`, fieldName, `) == 0 {`)
		g.P(`return `, g.errorsPkg.Use(), `.NotValidf("`, field.Name, ` is required")`)
		g.P(`}`)
		return
	}

	switch field.GetType() {
	case pb.FieldDescriptorProto_TYPE_BYTES:
		g.P(`if msg.`, fieldName, ` == nil || len(msg.`, fieldName, `) == 0 {`)
		g.P(`return `, g.errorsPkg.Use(), `.NotValidf("`, field.Name, ` is required")`)
		g.P(`}`)
	case pb.FieldDescriptorProto_TYPE_STRING:
		g.P(`if msg.`, fieldName, ` == "" {`)
		g.P(`return `, g.errorsPkg.Use(), `.NotValidf("`, field.Name, ` is required")`)
		g.P(`}`)
	case pb.FieldDescriptorProto_TYPE_MESSAGE:
		if gogoproto.IsNullable(field) {
			g.P(`if msg.`, fieldName, ` == nil {`)
			g.P(`return `, g.errorsPkg.Use(), `.NotValidf("`, field.Name, ` is required")`)
			g.P(`}`)
		}
	}

}

func (g *validation) generateSubMessageTest(msg *generator.Descriptor, field *pb.FieldDescriptorProto) {
	fieldName := g.gen.GetFieldName(msg, field)

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

	if field.IsRepeated() {
		g.P(`for _, value :=range msg.`, fieldName, `{`)
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
			g.P(`value := msg.`, fieldName)
			g.P(`if value != nil {`)
			g.P(`if err := `, casttyp, `.Validate(); err != nil {`)
			g.P(`return `, g.errorsPkg.Use(), `.Trace(err)`)
			g.P(`}`)
			g.P(`}`)
			g.P(`}`)
		} else {
			g.P(`{`)
			g.P(`value := msg.`, fieldName)
			g.P(`if (`, typ, `{}) != `, zeroValuer, ` {`)
			g.P(`if err := `, casttyp, `.Validate(); err != nil {`)
			g.P(`return `, g.errorsPkg.Use(), `.Trace(err)`)
			g.P(`}`)
			g.P(`}`)
			g.P(`}`)
		}
	}

}

func (g *validation) generateMinItemsTest(msg *generator.Descriptor, field *pb.FieldDescriptorProto, minItems int) {
	fieldName := g.gen.GetFieldName(msg, field)

	if !field.IsRepeated() {
		g.gen.Fail("limbo.minItems can only be used on repeated fields.")
	}

	g.P(`if len(msg.`, fieldName, `) < `, minItems, ` {`)
	g.P(`return `, g.errorsPkg.Use(), `.NotValidf("number of items in `, field.Name, ` (%q) is to small (minimum: `, minItems, `)", len(msg.`, fieldName, `))`)
	g.P(`}`)
}

func (g *validation) generateMaxItemsTest(msg *generator.Descriptor, field *pb.FieldDescriptorProto, maxItems int) {
	fieldName := g.gen.GetFieldName(msg, field)

	if !field.IsRepeated() {
		g.gen.Fail("limbo.maxItems can only be used on repeated fields.")
	}

	g.P(`if len(msg.`, fieldName, `) < `, maxItems, ` {`)
	g.P(`return `, g.errorsPkg.Use(), `.NotValidf("number of items in `, field.Name, ` (%q) is to small (maximum: `, maxItems, `)", len(msg.`, fieldName, `))`)
	g.P(`}`)
}

func (g *validation) generateMinLengthTest(msg *generator.Descriptor, field *pb.FieldDescriptorProto, minLength int) {
	fieldName := g.gen.GetFieldName(msg, field)

	if field.GetType() != pb.FieldDescriptorProto_TYPE_STRING {
		g.gen.Fail("limbo.minLength can only be used on string fields.")
	}

	if field.IsRepeated() {
		g.P(`for idx, value :=range msg.`, fieldName, `{`)
		g.P(`if len(value) < `, minLength, ` {`)
		g.P(`return `, g.errorsPkg.Use(), `.NotValidf("`, field.Name, `[%d] (%q) is to short (minimum length: `, minLength, `)", idx, value)`)
		g.P(`}`)
		g.P(`}`)
		return
	}

	g.P(`if len(msg.`, fieldName, `) < `, minLength, ` {`)
	g.P(`return `, g.errorsPkg.Use(), `.NotValidf("`, field.Name, ` (%q) is to short (minimum length: `, minLength, `)", msg.`, fieldName, `)`)
	g.P(`}`)

}

func (g *validation) generateMaxLengthTest(msg *generator.Descriptor, field *pb.FieldDescriptorProto, maxLength int) {
	fieldName := g.gen.GetFieldName(msg, field)

	if field.GetType() != pb.FieldDescriptorProto_TYPE_STRING {
		g.gen.Fail("limbo.maxLength can only be used on string fields.")
	}

	if field.IsRepeated() {
		g.P(`for idx, value :=range msg.`, fieldName, `{`)
		g.P(`if len(value) > `, maxLength, ` {`)
		g.P(`return `, g.errorsPkg.Use(), `.NotValidf("`, field.Name, `[%d] (%q) is to long (maximum length: `, maxLength, `)", idx, value)`)
		g.P(`}`)
		g.P(`}`)
		return
	}

	g.P(`if len(msg.`, fieldName, `) > `, maxLength, ` {`)
	g.P(`return `, g.errorsPkg.Use(), `.NotValidf("`, field.Name, ` (%q) is to long (maximum length: `, maxLength, `)", msg.`, fieldName, `)`)
	g.P(`}`)

}

func (g *validation) generatePatternTest(msg *generator.Descriptor, field *pb.FieldDescriptorProto, pattern, patternVar string) {
	fieldName := g.gen.GetFieldName(msg, field)

	if field.GetType() != pb.FieldDescriptorProto_TYPE_STRING {
		g.gen.Fail("limbo.pattern can only be used on string fields.")
	}

	if field.IsRepeated() {
		g.P(`for idx, value :=range msg.`, fieldName, `{`)
		g.P(`if !`, patternVar, `.MatchString(value) {`)
		g.P(`return `, g.errorsPkg.Use(), `.NotValidf("`, field.Name, `[%d] (%q) does not match pattern %s", idx, value, `, strconv.Quote(pattern), `)`)
		g.P(`}`)
		g.P(`}`)
		return
	}

	g.P(`if !`, patternVar, `.MatchString(msg.`, fieldName, `) {`)
	g.P(`return `, g.errorsPkg.Use(), `.NotValidf("`, field.Name, ` (%q) does not match pattern %s", msg.`, fieldName, `, `, strconv.Quote(pattern), `)`)
	g.P(`}`)

}
