package plugin

import "github.com/gogo/protobuf/protoc-gen-gogo/generator"

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
	for _, imp := range file.Dependency {
		if imp == "google/protobuf/timestamp.proto" {
			// fmt.Fprintf(os.Stderr, "g.gen.ImportMap=%v\n", g.gen.ImportMap)
			// fmt.Fprintf(os.Stderr, "g.gen.Pkg=%v\n", g.gen.Pkg)
			// fmt.Fprintf(os.Stderr, "g.gen.ImportPrefix=%v\n", g.gen.ImportPrefix)
			// fmt.Fprintf(os.Stderr, "g.gen.PackageImportPath=%v\n", g.gen.PackageImportPath)
			// fmt.Fprintf(os.Stderr, "g.gen.Param=%v\n", g.gen.Param)
			g.P(`var _ google_protobuf.Timestamp`)
		}
	}
}

func (g *builtin) GenerateImports(file *generator.FileDescriptor) {
}
