package gensql

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/limbo-services/protobuf/gogoproto"
	"github.com/limbo-services/protobuf/proto"
	pb "github.com/limbo-services/protobuf/protoc-gen-gogo/descriptor"
	"github.com/limbo-services/protobuf/protoc-gen-gogo/generator"

	"github.com/limbo-services/core/runtime/limbo"
)

func init() {
	generator.RegisterPlugin(new(gensql))
}

type gensql struct {
	gen *generator.Generator

	imports    generator.PluginImports
	sqlPkg     generator.Single
	runtimePkg generator.Single
	timePkg    generator.Single
	mysqlPkg   generator.Single

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
	g.runtimePkg = imp.NewImport("github.com/limbo-services/core/runtime/limbo")
	g.timePkg = imp.NewImport("time")
	g.mysqlPkg = imp.NewImport("github.com/go-sql-driver/mysql")

	var models []*generator.Descriptor

	for _, msg := range file.Messages() {
		model := limbo.GetModel(msg)
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
		g.generateStmt(file, msg)
		g.generateScanners(file, msg)
	}

	// phase 3
	for _, msg := range models {
		model := limbo.GetModel(msg)
		model.DeepColumn = nil
		model.DeepJoin = nil
		model.DeepScanner = nil
	}

}

// GenerateImports generates the import declaration for this file.
func (g *gensql) GenerateImports(file *generator.FileDescriptor) {
	g.imports.GenerateImports(file)
}

func unexport(s string) string        { return strings.ToLower(s[:1]) + s[1:] }
func prefixColumn(p, c string) string { return strings.TrimPrefix(p+"."+c, ".") }

func (g *gensql) populateMessage(file *generator.FileDescriptor, msg *generator.Descriptor) {
	model := limbo.GetModel(msg)
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
			model.Scanner = append(model.Scanner, &limbo.ScannerDescriptor{Fields: "*"})
		}
	}

	for _, scanner := range model.Scanner {
		scanner.MessageType = "." + file.GetPackage() + "." + msg.GetName()
	}

	for _, field := range msg.GetField() {
		if column := limbo.GetColumn(field); column != nil {
			column.MessageType = "." + file.GetPackage() + "." + msg.GetName()
			column.FieldName = field.GetName()
			if column.Name == "" {
				column.Name = field.GetName()
			}

			model.Column = append(model.Column, column)
		}

		if join := limbo.GetJoin(field); join != nil {
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

	sort.Sort(limbo.SortedColumnDescriptors(model.Column))
	sort.Sort(limbo.SortedJoinDescriptors(model.Join))
	sort.Sort(limbo.SortedScannerDescriptors(model.Scanner))
}

func (g *gensql) populateMessageDeep(msg *generator.Descriptor, stack []string) {
	model := limbo.GetModel(msg)

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

		fmodel := limbo.GetModel(fmsg)

		join.Table = fmodel.Table

		for _, fjoin := range fmodel.DeepJoin {
			fjoin = proto.Clone(fjoin).(*limbo.JoinDescriptor)
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
			fcolumn = proto.Clone(fcolumn).(*limbo.ColumnDescriptor)
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
			fscanner = proto.Clone(fscanner).(*limbo.ScannerDescriptor)
			if strings.ContainsRune(fscanner.Name, ':') {
				fscanner.Name = join.FieldName + "." + fscanner.Name
			} else {
				fscanner.Name = join.FieldName + ":" + fscanner.Name
			}

			for i, fjoin := range fscanner.Join {
				fjoin = proto.Clone(fjoin).(*limbo.JoinDescriptor)
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
				fcolumn = proto.Clone(fcolumn).(*limbo.ColumnDescriptor)
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

func (g *gensql) populateScanner(msg *generator.Descriptor, scanner *limbo.ScannerDescriptor) {
	if len(scanner.Column) > 0 {
		return
	}

	var (
		ops   = strings.Split(scanner.Fields, ",")
		queue = ops
		model = limbo.GetModel(msg)
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

	selected := make(map[string]*limbo.ColumnDescriptor)

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

	var columns []*limbo.ColumnDescriptor

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

	sort.Sort(limbo.SortedColumnDescriptors(scanner.Column))
	sort.Sort(limbo.SortedJoinDescriptors(scanner.Join))
}

func (g *gensql) generateScanners(file *generator.FileDescriptor, message *generator.Descriptor) {
	model := limbo.GetModel(message)
	for _, scanner := range model.Scanner {
		g.generateScanner(file, message, scanner)
	}
}

func (g *gensql) generateQueryPrefix(message *generator.Descriptor, scanner *limbo.ScannerDescriptor) string {
	var (
		buf   bytes.Buffer
		model = limbo.GetModel(message)
	)

	buf.WriteString("SELECT")

	for i, column := range scanner.Column {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.WriteByte(' ')
		if column.JoinedWith == "" {
			buf.WriteString(model.Table)
			buf.WriteByte('.')
		}
		buf.WriteString(column.Name)
	}

	buf.WriteString(" FROM ")
	buf.WriteString(model.Table)

	for _, join := range scanner.Join {
		buf.WriteString(" LEFT JOIN ")
		buf.WriteString(join.Table)
		buf.WriteString(" AS ")
		buf.WriteString(join.Name)
		buf.WriteString(" ON ")
		buf.WriteString(join.Name)
		buf.WriteByte('.')
		buf.WriteString(join.ForeignKey)
		buf.WriteString(" = ")
		if join.JoinedWith != "" {
			buf.WriteString(scanner.LookupJoin(join.JoinedWith).Name)
		} else {
			buf.WriteString(model.Table)
		}
		buf.WriteByte('.')
		buf.WriteString(join.Key)
	}

	return buf.String()
}

func (g *gensql) generateScanner(file *generator.FileDescriptor, message *generator.Descriptor, scanner *limbo.ScannerDescriptor) {
	scannerFuncName := `scan_` + message.GetName()
	if scanner.Name != "" {
		scannerFuncName += `_` + scanner.Name
	}

	joins := map[string]int{}

	g.P(``)
	g.P(`const `, scannerFuncName, `SQL = `, strconv.Quote(g.generateQueryPrefix(message, scanner)))
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
			if field.GetTypeName() == ".google.protobuf.Timestamp" {
				g.P(`b`, i, ` `, g.mysqlPkg.Use(), `.NullTime`)
			} else if limbo.IsGoSQLValuer(g.objectNamed(field.GetTypeName()).(*generator.Descriptor)) {
				g.P(`b`, i, ` Null`, g.typeName(field.GetTypeName()))
			} else {
				g.P(`b`, i, ` []byte`)
				g.P(`m`, i, ` `, g.typeName(field.GetTypeName()))
			}
		default:
			panic("unsuported type: " + field.GetType().String())
		}
	}
	for i, join := range scanner.Join {
		m := g.models[join.MessageType]
		field := m.GetFieldDescriptor(lastField(join.FieldName))
		joins[join.FieldName] = i
		g.P(`j`, i, ` `, g.typeName(field.GetTypeName()))
		g.P(`j`, i, `Valid bool`)
	}
	g.P(`)`)
	g.P(`err := scanFunc(`)
	for i := range scanner.Column {
		g.P(`&b`, i, `,`)
	}
	g.P(`)`)
	g.P(`if err!=nil { return err }`)
	for i, column := range scanner.Column {
		var (
			m     = g.models[column.MessageType]
			field = m.GetFieldDescriptor(lastField(column.FieldName))
			valid string
			dst   = "dst"
		)

		if column.JoinedWith != "" {
			dst = fmt.Sprintf("j%d", joins[column.JoinedWith])
		}

		switch field.GetType() {
		case pb.FieldDescriptorProto_TYPE_MESSAGE:
			if field.GetTypeName() == ".google.protobuf.Timestamp" {
				valid = fmt.Sprintf(`b%d.Valid`, i)
			} else if limbo.IsGoSQLValuer(g.objectNamed(field.GetTypeName()).(*generator.Descriptor)) {
				valid = fmt.Sprintf(`b%d.Valid`, i)
			} else {
				valid = fmt.Sprintf(`b%d != nil`, i)
			}
		case pb.FieldDescriptorProto_TYPE_BYTES:
			valid = fmt.Sprintf(`b%d != nil`, i)
		default:
			valid = fmt.Sprintf(`b%d.Valid`, i)
		}

		fieldName := g.gen.GetFieldName(m, field)

		g.P(`if `, valid, ` {`)
		if column.JoinedWith != "" {
			g.P(dst, `Valid = true`)
		}
		switch field.GetType() {
		case pb.FieldDescriptorProto_TYPE_BOOL:
			g.P(dst, `.`, fieldName, ` = b`, i, `.Bool`)
		case pb.FieldDescriptorProto_TYPE_DOUBLE:
			g.P(dst, `.`, fieldName, ` = float32(b`, i, `.Float64)`)
		case pb.FieldDescriptorProto_TYPE_FLOAT:
			g.P(dst, `.`, fieldName, ` = float32(b`, i, `.Float64)`)
		case pb.FieldDescriptorProto_TYPE_FIXED32,
			pb.FieldDescriptorProto_TYPE_UINT32:
			g.P(dst, `.`, fieldName, ` = uint32(b`, i, `.Int64)`)
		case pb.FieldDescriptorProto_TYPE_FIXED64,
			pb.FieldDescriptorProto_TYPE_UINT64:
			g.P(dst, `.`, fieldName, ` = uint64(b`, i, `.Int64)`)
		case pb.FieldDescriptorProto_TYPE_SFIXED32,
			pb.FieldDescriptorProto_TYPE_INT32,
			pb.FieldDescriptorProto_TYPE_SINT32:
			g.P(dst, `.`, fieldName, ` = int32(b`, i, `.Int64)`)
		case pb.FieldDescriptorProto_TYPE_SFIXED64,
			pb.FieldDescriptorProto_TYPE_INT64,
			pb.FieldDescriptorProto_TYPE_SINT64:
			g.P(dst, `.`, fieldName, ` = int64(b`, i, `.Int64)`)
		case pb.FieldDescriptorProto_TYPE_BYTES:
			g.P(dst, `.`, fieldName, ` = b`, i, ``)
		case pb.FieldDescriptorProto_TYPE_STRING:
			g.P(dst, `.`, fieldName, ` = b`, i, `.String`)
		case pb.FieldDescriptorProto_TYPE_MESSAGE:
			if field.GetTypeName() == ".google.protobuf.Timestamp" {
				if gogoproto.IsNullable(field) {
					g.P(dst, `.`, fieldName, ` = &b`, i, `.Time`)
				} else {
					g.P(dst, `.`, fieldName, ` = b`, i, `.Time`)
				}
			} else if limbo.IsGoSQLValuer(g.objectNamed(field.GetTypeName()).(*generator.Descriptor)) {
				if gogoproto.IsNullable(field) {
					g.P(dst, `.`, fieldName, ` = &b`, i, `.`, g.typeName(field.GetTypeName()))
				} else {
					g.P(dst, `.`, fieldName, ` = b`, i, `.`, g.typeName(field.GetTypeName()))
				}
			} else {
				g.P(`err := m`, i, `.Unmarshal(b`, i, `)`)
				g.P(`if err!=nil { return err }`)
				if gogoproto.IsNullable(field) {
					g.P(dst, `.`, fieldName, ` = &m`, i, ``)
				} else {
					g.P(dst, `.`, fieldName, ` = m`, i, ``)
				}
			}
		}
		g.P(`}`)
	}
	for i := len(scanner.Join) - 1; i >= 0; i-- {
		var (
			join  = scanner.Join[i]
			m     = g.models[join.MessageType]
			field = m.GetFieldDescriptor(lastField(join.FieldName))
			dst   = "dst"
		)

		if join.JoinedWith != "" {
			dst = fmt.Sprintf("j%d", joins[join.JoinedWith])
		}

		fieldName := g.gen.GetFieldName(m, field)

		g.P(`if j`, i, `Valid {`)
		if gogoproto.IsNullable(field) {
			g.P(dst, `.`, fieldName, ` = &j`, i, ``)
		} else {
			g.P(dst, `.`, fieldName, ` = j`, i, ``)
		}
		g.P(`}`)
	}
	g.P(`return nil`)
	g.P(`}`)
	g.P(``)
}

func (g *gensql) generateStmt(file *generator.FileDescriptor, message *generator.Descriptor) {
	model := limbo.GetModel(message)

	g.P(`type `, message.Name, `StmtBuilder interface {`)
	g.P(`Prepare(scanner string, query string) `, message.Name, `Stmt`)
	g.P(`Err() error`)
	g.P(`}`)

	g.P(`type `, message.Name, `Stmt interface {`)
	g.P(`Exec(args ... interface{}) (`, g.sqlPkg.Use(), `.Result, error)`)
	g.P(`QueryRow(args ... interface{}) (`, message.Name, `Row)`)
	g.P(`Query(args ... interface{}) (`, message.Name, `Rows, error)`)
	g.P(`SelectSlice(dst []*`, message.Name, `, args ... interface{}) ([]*`, message.Name, `, error)`)
	g.P(`}`)

	g.P(`type `, message.Name, `Row interface {`)
	g.P(`Scan(out *`, message.Name, `)  error`)
	g.P(`}`)

	g.P(`type `, message.Name, `Rows interface {`)
	g.P(`Close() error`)
	g.P(`Next() bool`)
	g.P(`Err() error`)
	g.P(`Scan(out *`, message.Name, `)  error`)
	g.P(`}`)

	g.P(`type `, unexport(*message.Name), `StmtBuilder struct {`)
	g.P(`db *`, g.sqlPkg.Use(), `.DB`)
	g.P(`err error`)
	g.P(`}`)

	g.P(`type `, unexport(*message.Name), `Stmt struct {`)
	g.P(`stmt *`, g.sqlPkg.Use(), `.Stmt`)
	g.P(`scanner func(func(...interface{}) error, *`, message.Name, `) error`)
	g.P(`}`)

	g.P(`type `, unexport(*message.Name), `Row struct {`)
	g.P(`row *`, g.sqlPkg.Use(), `.Row`)
	g.P(`scanner func(func(...interface{}) error, *`, message.Name, `) error`)
	g.P(`}`)

	g.P(`type `, unexport(*message.Name), `Rows struct {`)
	g.P(`*`, g.sqlPkg.Use(), `.Rows`)
	g.P(`scanner func(func(...interface{}) error, *`, message.Name, `) error`)
	g.P(`}`)

	g.P(`func New`, message.Name, `StmtBuilder(db *`, g.sqlPkg.Use(), `.DB) `, message.Name, `StmtBuilder {`)
	g.P(`return &`, unexport(*message.Name), `StmtBuilder{db: db}`)
	g.P(`}`)

	g.P(`func (b *`, unexport(*message.Name), `StmtBuilder) Prepare(scanner string, query string) (`, message.Name, `Stmt) {`)
	g.P(`if b.err != nil { return nil }`)
	g.P(`var scannerFunc func(func(...interface{}) error, *`, message.Name, `) error`)
	g.P(`switch scanner {`)
	for _, scanner := range model.Scanner {
		scannerFuncName := `scan_` + message.GetName()
		if scanner.Name != "" {
			scannerFuncName += `_` + scanner.Name
		}

		g.P(`case `, strconv.Quote(scanner.Name), `:`)
		g.P(`query = `, scannerFuncName, `SQL + " " + query`)
		g.P(`scannerFunc = `, scannerFuncName)
	}
	g.P(`default:`)
	g.P(`if b.err == nil { b.err = fmt.Errorf("unknown scanner: %s", scanner) }`)
	g.P(`}`)
	g.P(`stmt, err := b.db.Prepare(`, g.runtimePkg.Use(), `.CleanSQL(query))`)
	g.P(`if err != nil { if b.err == nil { b.err = err } }`)
	g.P(`return &`, unexport(*message.Name), `Stmt{stmt: stmt, scanner: scannerFunc}`)
	g.P(`}`)

	g.P(`func (b *`, unexport(*message.Name), `StmtBuilder) Err() (error) {`)
	g.P(`return b.err`)
	g.P(`}`)

	g.P(`func (s *`, unexport(*message.Name), `Stmt) QueryRow(args ... interface{}) (`, message.Name, `Row) {`)
	g.P(`row := s.stmt.QueryRow(args...)`)
	g.P(`return &`, unexport(*message.Name), `Row{row: row, scanner: s.scanner}`)
	g.P(`}`)

	g.P(`func (s *`, unexport(*message.Name), `Stmt) Query(args ... interface{}) (`, message.Name, `Rows, error) {`)
	g.P(`rows, err := s.stmt.Query(args...)`)
	g.P(`if err != nil { return nil, err }`)
	g.P(`return &`, unexport(*message.Name), `Rows{Rows: rows, scanner: s.scanner}, nil`)
	g.P(`}`)

	g.P(`func (s *`, unexport(*message.Name), `Stmt) Exec(args ... interface{}) (`, g.sqlPkg.Use(), `.Result, error) {`)
	g.P(`return s.stmt.Exec(args...)`)
	g.P(`}`)

	g.P(`func (s *`, unexport(*message.Name), `Stmt) SelectSlice(dst []*`, message.Name, `, args ... interface{}) ([]*`, message.Name, `, error) {`)
	g.P(`rows, err := s.Query(args...)`)
	g.P(`if err != nil { return nil, err }`)
	g.P(`defer rows.Close()`)
	g.P(`for rows.Next() {`)
	g.P(`var x = &`, message.Name, `{}`)
	g.P(`err := rows.Scan(x)`)
	g.P(`if err != nil { return nil, err }`)
	g.P(`dst = append(dst, x)`)
	g.P(`}`)
	g.P(`err = rows.Err()`)
	g.P(`if err != nil { return nil, err }`)
	g.P(`return dst, nil`)
	g.P(`}`)

	g.P(`func (r *`, unexport(*message.Name), `Row) Scan(out *`, message.Name, `) error {`)
	g.P(`return r.scanner(r.row.Scan, out)`)
	g.P(`}`)

	g.P(`func (r *`, unexport(*message.Name), `Rows) Scan(out *`, message.Name, `) error {`)
	g.P(`return r.scanner(r.Rows.Scan, out)`)
	g.P(`}`)
}

func lastField(s string) string {
	if idx := strings.IndexByte(s, '.'); idx >= 0 {
		return s[idx+1:]
	}
	return s
}
