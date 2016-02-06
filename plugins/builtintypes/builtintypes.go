package plugin

import "github.com/limbo-services/protobuf/protoc-gen-gogo/generator"

func init() {
	generator.RegisterPlugin(new(builtin))
}

type builtin struct {
	gen *generator.Generator
}

func (g *builtin) Name() string {
	return "builtintypes"
}

func (g *builtin) Init(gen *generator.Generator) {
	g.gen = gen
}

func (g *builtin) P(args ...interface{}) { g.gen.P(args...) }

func (g *builtin) Generate(file *generator.FileDescriptor) {
}

func (g *builtin) GenerateImports(file *generator.FileDescriptor) {
}
