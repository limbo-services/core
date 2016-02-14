package middelware

import "github.com/limbo-services/protobuf/protoc-gen-gogo/generator"

func init() {
	generator.RegisterPlugin(new(middelware))
}

type middelware struct {
	gen *generator.Generator

	imports generator.PluginImports
}

func (g *middelware) Name() string {
	return "middelware"
}

func (g *middelware) Init(gen *generator.Generator) {
	g.gen = gen
}

func (g *middelware) Generate(file *generator.FileDescriptor) {
	g.imports = generator.NewPluginImports(g.gen)
	runtimePkg := g.imports.NewImport("github.com/limbo-services/core/runtime/limbo")

	for _, service := range file.FileDescriptorProto.Service {
		descName := "_" + service.GetName() + "_serviceDesc"
		g.gen.AddInitf("%s.Wrap(&%s)", runtimePkg.Use(), descName)
	}
}

func (g *middelware) GenerateImports(file *generator.FileDescriptor) {
	g.imports.GenerateImports(file)
}
