package gensql

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/fd/featherhead/tools/runtime/sql"
	"github.com/gogo/protobuf/proto"
	pb "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
)

func init() {
	generator.RegisterPlugin(new(gensql))
}

type gensql struct {
	gen *generator.Generator

	imports generator.PluginImports
	sqlPkg  generator.Single

	models map[string]*generator.Descriptor
}

func (g *gensql) Name() string {
	return "sql"
}

// Init initializes the plugin.
func (g *gensql) Init(gen *generator.Generator) {
	g.gen = gen
	g.models = make(map[string]*generator.Descriptor)
}

// Given a type name defined in a .proto, return its object.
// Also record that we're using it, to guarantee the associated import.
func (g *gensql) objectNamed(name string) generator.Object {
	g.gen.RecordTypeUse(name)
	return g.gen.ObjectNamed(name)
}

// Given a type name defined in a .proto, return its name as we will print it.
func (g *gensql) typeName(str string) string {
	return g.gen.TypeName(g.objectNamed(str))
}

// P forwards to g.gen.P.
func (g *gensql) P(args ...interface{}) { g.gen.P(args...) }

// Generate generates code for the services in the given file.
func (g *gensql) Generate(file *generator.FileDescriptor) {
	imp := generator.NewPluginImports(g.gen)
	g.imports = imp
	g.sqlPkg = imp.NewImport("database/sql")

	var models []*generator.Descriptor

	for _, msg := range file.Messages() {
		model := sql.GetModel(msg)
		if model == nil {
			continue
		}
		g.models["."+file.GetPackage()+"."+msg.GetName()] = msg
		models = append(models, msg)
	}

	if len(models) == 0 {
		return
	}

	// phase 1
	for _, msg := range models {
		g.populateMessage(file, msg)
	}

	// phase 2
	for _, msg := range models {
		g.populateMessageDeep(msg, nil)
	}

	for _, msg := range models {
		g.generateScanners(file, msg)
	}

	// phase 3
	for _, msg := range models {
		model := sql.GetModel(msg)
		model.DeepColumn = nil
		model.DeepJoin = nil
		model.DeepScanner = nil
	}

	// phase 4
	for _, msg := range models {
		g.populateScanners(msg)
	}

	// for i, msg := range file.Messages() {
	// 	g.generateScanners(file, msg, i)
	// }

}

// GenerateImports generates the import declaration for this file.
func (g *gensql) GenerateImports(file *generator.FileDescriptor) {
	g.imports.GenerateImports(file)
}

func unexport(s string) string        { return strings.ToLower(s[:1]) + s[1:] }
func prefixColumn(p, c string) string { return strings.TrimPrefix(p+"."+c, ".") }

func (g *gensql) populateMessage(file *generator.FileDescriptor, msg *generator.Descriptor) {
	model := sql.GetModel(msg)
	model.MessageType = "." + file.GetPackage() + "." + msg.GetName()

	{ // default scanner
		var found bool
		for _, scanner := range model.Scanner {
			if scanner.Name == "" {
				found = true
				break
			}
		}
		if !found {
			model.Scanner = append(model.Scanner, &sql.ScannerDescriptor{Fields: "*"})
		}
	}

	for _, scanner := range model.Scanner {
		scanner.MessageType = "." + file.GetPackage() + "." + msg.GetName()
	}

	for _, field := range msg.GetField() {
		if column := sql.GetColumn(field); column != nil {
			column.MessageType = "." + file.GetPackage() + "." + msg.GetName()
			column.FieldName = field.GetName()
			if column.Name == "" {
				column.Name = field.GetName()
			}

			model.Column = append(model.Column, column)
		}

		if join := sql.GetJoin(field); join != nil {
			if field.GetType() != pb.FieldDescriptorProto_TYPE_MESSAGE {
				g.gen.Fail(field.GetName(), "in", msg.GetName(), "must be a message")
			}

			join.MessageType = "." + file.GetPackage() + "." + msg.GetName()
			join.FieldName = field.GetName()
			join.ForeignMessageType = field.GetTypeName()

			if join.Name == "" {
				join.Name = field.GetName()
			}

			if join.Key == "" {
				join.Key = field.GetName() + "_id"
			}

			if join.ForeignKey == "" {
				join.ForeignKey = "id"
			}

			model.Join = append(model.Join, join)
		}
	}

	sort.Sort(sql.SortedColumnDescriptors(model.Column))
	sort.Sort(sql.SortedJoinDescriptors(model.Join))
	sort.Sort(sql.SortedScannerDescriptors(model.Scanner))
}

func (g *gensql) populateMessageDeep(msg *generator.Descriptor, stack []string) {
	model := sql.GetModel(msg)

	for _, i := range stack {
		if i == model.MessageType {
			g.gen.Fail("models cannot have join loops")
		}
	}
	stack = append(stack, model.MessageType)

	if len(model.DeepColumn) > 0 {
		return
	}

	model.DeepColumn = model.Column
	model.DeepJoin = model.Join

	for _, join := range model.Join {
		fmsg := g.models[join.ForeignMessageType]
		if fmsg == nil {
			g.gen.Fail(model.MessageType, ":", join.ForeignMessageType, "is not a model")
		}

		g.populateMessageDeep(fmsg, stack)

		fmodel := sql.GetModel(fmsg)

		for _, fjoin := range fmodel.DeepJoin {
			fjoin = proto.Clone(fjoin).(*sql.JoinDescriptor)
			fjoin.FieldName = join.FieldName + "." + fjoin.FieldName
			fjoin.Name = join.Name + "_" + fjoin.Name
			if fjoin.JoinedWith == "" {
				fjoin.JoinedWith = join.FieldName
			} else {
				fjoin.JoinedWith = join.FieldName + "." + fjoin.JoinedWith
			}
			model.DeepJoin = append(model.DeepJoin, fjoin)
		}

		for _, fcolumn := range fmodel.DeepColumn {
			fcolumn = proto.Clone(fcolumn).(*sql.ColumnDescriptor)
			fcolumn.FieldName = join.FieldName + "." + fcolumn.FieldName
			if strings.ContainsRune(fcolumn.Name, '.') {
				fcolumn.Name = join.Name + "_" + fcolumn.Name
			} else {
				fcolumn.Name = join.Name + "." + fcolumn.Name
			}
			if fcolumn.JoinedWith == "" {
				fcolumn.JoinedWith = join.FieldName
			} else {
				fcolumn.JoinedWith = join.FieldName + "." + fcolumn.JoinedWith
			}
			model.DeepColumn = append(model.DeepColumn, fcolumn)
		}

		for _, fscanner := range fmodel.DeepScanner {
			fscanner = proto.Clone(fscanner).(*sql.ScannerDescriptor)
			if strings.ContainsRune(fscanner.Name, ':') {
				fscanner.Name = join.FieldName + "." + fscanner.Name
			} else {
				fscanner.Name = join.FieldName + ":" + fscanner.Name
			}

			for i, fjoin := range fscanner.Join {
				fjoin = proto.Clone(fjoin).(*sql.JoinDescriptor)
				fjoin.FieldName = join.FieldName + "." + fjoin.FieldName
				fjoin.Name = join.Name + "_" + fjoin.Name
				if fjoin.JoinedWith == "" {
					fjoin.JoinedWith = join.FieldName
				} else {
					fjoin.JoinedWith = join.FieldName + "." + fjoin.JoinedWith
				}
				fscanner.Join[i] = fjoin
			}

			for i, fcolumn := range fscanner.Column {
				fcolumn = proto.Clone(fcolumn).(*sql.ColumnDescriptor)
				fcolumn.FieldName = join.FieldName + "." + fcolumn.FieldName
				if strings.ContainsRune(fcolumn.Name, '.') {
					fcolumn.Name = join.Name + "_" + fcolumn.Name
				} else {
					fcolumn.Name = join.Name + "." + fcolumn.Name
				}
				if fcolumn.JoinedWith == "" {
					fcolumn.JoinedWith = join.FieldName
				} else {
					fcolumn.JoinedWith = join.FieldName + "." + fcolumn.JoinedWith
				}
				fscanner.Column[i] = fcolumn
			}

			model.DeepScanner = append(model.DeepScanner, fscanner)
		}
	}

	for _, scanner := range model.Scanner {
		g.populateScanner(msg, scanner)
		model.DeepScanner = append(model.DeepScanner, scanner)
	}
}

func (g *gensql) populateScanners(msg *generator.Descriptor) {
	model := sql.GetModel(msg)
	data, _ := json.MarshalIndent(model, "", "  ")
	fmt.Fprintln(os.Stderr, string(data))
}

func (g *gensql) populateScanner(msg *generator.Descriptor, scanner *sql.ScannerDescriptor) {
	if len(scanner.Column) > 0 {
		return
	}

	var (
		ops   = strings.Split(scanner.Fields, ",")
		queue = ops
		model = sql.GetModel(msg)
	)

	queue = ops
	ops = nil
	for _, op := range queue {
		op = strings.TrimSpace(op)

		if op == "*" {
			for _, column := range model.Column {
				ops = append(ops, column.FieldName)
			}
			for _, join := range model.Join {
				ops = append(ops, join.FieldName)
			}
		} else {
			ops = append(ops, op)
		}
	}

	queue = ops
	ops = nil
	for _, op := range queue {
		var (
			found    bool
			isRemove bool
			name     = op
		)

		if strings.HasPrefix(op, "-") {
			name = op[1:]
			isRemove = true
		}

		if !strings.ContainsRune(name, ':') {
			name += ":"
		}

		for _, scanner := range model.DeepScanner {
			if scanner.Name == name {
				found = true
				for _, column := range scanner.Column {
					if !isRemove {
						ops = append(ops, column.FieldName)
					} else {
						ops = append(ops, "-"+column.FieldName)
					}
				}
				break
			}
		}

		if !found {
			ops = append(ops, op)
		}
	}

	selected := make(map[string]*sql.ColumnDescriptor)

	queue = ops
	ops = nil
	fmt.Fprintf(os.Stderr, "%q => %q\n", scanner.Name, queue)
	for _, op := range queue {
		var (
			found    bool
			isRemove bool
			name     = op
		)

		if strings.HasPrefix(op, "-") {
			name = op[1:]
			isRemove = true
		}

		for _, column := range model.DeepColumn {
			if column.FieldName == name {
				found = true
				if !isRemove {
					selected[name] = column
					ops = append(ops, op)
				} else {
					selected[name] = nil
				}
				break
			}
		}

		if !found {
			g.gen.Fail("unknown column", name)
		}
	}

	var columns []*sql.ColumnDescriptor

	queue = ops
	ops = nil
	for _, op := range queue {
		column := selected[op]
		if column != nil {
			columns = append(columns, column)
			selected[op] = nil
		}
	}

	scanner.Column = columns

	joinQueue := []string{}
	seenJoin := make(map[string]bool)
	for _, column := range scanner.Column {
		if column.JoinedWith == "" {
			continue
		}

		joinQueue = append(joinQueue, column.JoinedWith)
	}

	for len(joinQueue) > 0 {
		joinName := joinQueue[0]
		joinQueue = joinQueue[1:]
		if seenJoin[joinName] {
			continue
		}
		seenJoin[joinName] = true

		var found bool
		for _, join := range model.DeepJoin {
			if joinName != join.Name {
				continue
			}

			found = true
			scanner.Join = append(scanner.Join, join)
			if join.JoinedWith != "" {
				joinQueue = append(joinQueue, join.JoinedWith)
			}
			break
		}

		if !found {
			g.gen.Fail("unknown join", joinName)
		}
	}
}

func (g *gensql) generateScanners(file *generator.FileDescriptor, message *generator.Descriptor) {
	model := sql.GetModel(message)
	for _, scanner := range model.Scanner {
		g.generateScanner(file, message, scanner)
	}
}

func (g *gensql) generateScanner(file *generator.FileDescriptor, message *generator.Descriptor, scanner *sql.ScannerDescriptor) {
	scannerFuncName := `scan_` + message.GetName()
	if scanner.Name != "" {
		scannerFuncName += `_` + scanner.Name
	}

	g.P(`func `, scannerFuncName, `(scanFunc func(...interface{})error, dst *`, message.Name, `) error {`)
	g.P(`var (`)
	for i, column := range scanner.Column {
		m := g.models[column.MessageType]
		field := m.GetFieldDescriptor(lastField(column.FieldName))
		switch field.GetType() {
		case pb.FieldDescriptorProto_TYPE_BOOL:
			g.P(`b`, i, ` `, g.sqlPkg.Use(), `.NullBool`)
		case pb.FieldDescriptorProto_TYPE_DOUBLE:
			g.P(`b`, i, ` `, g.sqlPkg.Use(), `.NullFloat64`)
		case pb.FieldDescriptorProto_TYPE_FLOAT:
			g.P(`b`, i, ` `, g.sqlPkg.Use(), `.NullFloat64`)
		case pb.FieldDescriptorProto_TYPE_FIXED32,
			pb.FieldDescriptorProto_TYPE_UINT32:
			g.P(`b`, i, ` `, g.sqlPkg.Use(), `.NullInt64`)
		case pb.FieldDescriptorProto_TYPE_FIXED64,
			pb.FieldDescriptorProto_TYPE_UINT64:
			g.P(`b`, i, ` `, g.sqlPkg.Use(), `.NullInt64`)
		case pb.FieldDescriptorProto_TYPE_SFIXED32,
			pb.FieldDescriptorProto_TYPE_INT32,
			pb.FieldDescriptorProto_TYPE_SINT32:
			g.P(`b`, i, ` `, g.sqlPkg.Use(), `.NullInt64`)
		case pb.FieldDescriptorProto_TYPE_SFIXED64,
			pb.FieldDescriptorProto_TYPE_INT64,
			pb.FieldDescriptorProto_TYPE_SINT64:
			g.P(`b`, i, ` `, g.sqlPkg.Use(), `.NullInt64`)
		case pb.FieldDescriptorProto_TYPE_BYTES:
			g.P(`b`, i, ` []byte`)
		case pb.FieldDescriptorProto_TYPE_STRING:
			g.P(`b`, i, ` `, g.sqlPkg.Use(), `.NullString`)
		case pb.FieldDescriptorProto_TYPE_MESSAGE:
			g.P(`b`, i, ` []byte`)
			g.P(`m`, i, ` `, g.typeName(field.GetTypeName()))
		default:
			panic("unsupoorted type: " + field.GetType().String())
		}
	}
	g.P(`)`)
	g.P(`err := scanFunc(`)
	for i := range scanner.Column {
		g.P(`&b`, i, `,`)
	}
	g.P(`)`)
	g.P(`if err!=nil { return err }`)
	for i, column := range scanner.Column {
		m := g.models[column.MessageType]
		field := m.GetFieldDescriptor(lastField(column.FieldName))
		var valid string

		switch field.GetType() {
		case pb.FieldDescriptorProto_TYPE_MESSAGE,
			pb.FieldDescriptorProto_TYPE_BYTES:
			valid = fmt.Sprintf(`b%d != nil`, i)
		default:
			valid = fmt.Sprintf(`b%d.Valid`, i)
		}

		g.P(`if `, valid, ` {`)
		switch field.GetType() {
		case pb.FieldDescriptorProto_TYPE_BOOL:
			g.P(`b`, i, `.Value`)
		case pb.FieldDescriptorProto_TYPE_DOUBLE:
			g.P(`float32(b`, i, `.Value)`)
		case pb.FieldDescriptorProto_TYPE_FLOAT:
			g.P(`float32(b`, i, `.Value)`)
		case pb.FieldDescriptorProto_TYPE_FIXED32,
			pb.FieldDescriptorProto_TYPE_UINT32:
			g.P(`uint32(b`, i, `.Value)`)
		case pb.FieldDescriptorProto_TYPE_FIXED64,
			pb.FieldDescriptorProto_TYPE_UINT64:
			g.P(`uint64(b`, i, `.Value)`)
		case pb.FieldDescriptorProto_TYPE_SFIXED32,
			pb.FieldDescriptorProto_TYPE_INT32,
			pb.FieldDescriptorProto_TYPE_SINT32:
			g.P(`int32(b`, i, `.Value)`)
		case pb.FieldDescriptorProto_TYPE_SFIXED64,
			pb.FieldDescriptorProto_TYPE_INT64,
			pb.FieldDescriptorProto_TYPE_SINT64:
			g.P(`int64(b`, i, `.Value)`)
		case pb.FieldDescriptorProto_TYPE_BYTES:
			g.P(`b`, i, ``)
		case pb.FieldDescriptorProto_TYPE_STRING:
			g.P(`b`, i, `.Value`)
		case pb.FieldDescriptorProto_TYPE_MESSAGE:
			g.P(`err := m`, i, `.Unmarshal(b`, i, `)`)
			g.P(`if err!=nil { return err }`)
		}
		g.P(`}`)
	}
	g.P(`}`)
	g.P(``)
}

func lastField(s string) string {
	if idx := strings.IndexByte(s, '.'); idx >= 0 {
		return s[idx+1:]
	}
	return s
}
