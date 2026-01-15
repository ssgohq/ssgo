package gorm

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/ssgohq/ssgo/internal/dbparser"
	"github.com/ssgohq/ssgo/tool/internal/generator/naming"
)

// modelData holds data for model template rendering
type modelData struct {
	PackageName string
	Imports     []string
	StructName  string
	TableName   string
	Comment     string
	Fields      []modelField
	PrimaryKey  *modelPrimaryKey
	SoftDelete  bool
}

type modelField struct {
	Name       string
	Type       string
	Column     string
	GormTags   []string
	Comment    string
	JSONName   string
	IsNullable bool
}

type modelPrimaryKey struct {
	Fields []string
}

// generateModel generates a single model file
func (g *Generator) generateModel(dir string, table *dbparser.Table) error {
	data := g.buildModelData(table)

	// Render template
	content, err := g.renderModelTemplate(data)
	if err != nil {
		return err
	}

	// Write file
	filename := filepath.Join(dir, naming.ToSnakeCase(table.Name)+"_gen.go")
	g.logVerbose("  Generating model: %s\n", filename)

	return os.WriteFile(filename, content, 0o644)
}

// buildModelData converts table schema to template data
func (g *Generator) buildModelData(table *dbparser.Table) *modelData {
	imports := map[string]bool{}
	var fields []modelField

	// Check if table has soft delete (deleted_at column)
	hasSoftDelete := false
	for _, col := range table.Columns {
		if col.Name == "deleted_at" && g.opts.SoftDelete {
			hasSoftDelete = true
			break
		}
	}

	// If using GORM soft delete, we need gorm.DeletedAt type
	if hasSoftDelete {
		imports["gorm.io/gorm"] = true
	}

	for _, col := range table.Columns {
		// Skip deleted_at if we're using gorm.Model soft delete
		if col.Name == "deleted_at" && hasSoftDelete {
			continue
		}
		field := g.buildModelField(col, imports)
		fields = append(fields, field)
	}

	// Build primary key info
	var pk *modelPrimaryKey
	if table.PrimaryKey != nil {
		pkFields := make([]string, len(table.PrimaryKey.Columns))
		for i, colName := range table.PrimaryKey.Columns {
			pkFields[i] = naming.ToPascalCase(colName)
		}
		pk = &modelPrimaryKey{Fields: pkFields}
	}

	// Sort imports
	var importList []string
	for imp := range imports {
		importList = append(importList, imp)
	}
	sort.Strings(importList)

	return &modelData{
		PackageName: g.opts.ModelPackage,
		Imports:     importList,
		StructName:  naming.ToPascalCase(naming.Singularize(table.Name)),
		TableName:   table.Name,
		Comment:     table.Comment,
		Fields:      fields,
		PrimaryKey:  pk,
		SoftDelete:  hasSoftDelete,
	}
}

// buildModelField converts column to field data
func (g *Generator) buildModelField(col *dbparser.Column, imports map[string]bool) modelField {
	// Collect imports from Go type
	goType := col.GoType
	if strings.Contains(goType, "time.") {
		imports["time"] = true
	}
	if strings.Contains(goType, "json.") {
		imports["encoding/json"] = true
	}
	if strings.Contains(goType, "uuid.") {
		imports["github.com/google/uuid"] = true
	}
	if strings.Contains(goType, "decimal.") {
		imports["github.com/shopspring/decimal"] = true
	}
	if strings.Contains(goType, "sql.") {
		imports["database/sql"] = true
	}
	if strings.Contains(goType, "datatypes.") {
		imports["gorm.io/datatypes"] = true
	}

	// Build gorm tags
	gormTags := g.buildGormTags(col)

	return modelField{
		Name:       naming.ToPascalCase(col.Name),
		Type:       goType,
		Column:     col.Name,
		GormTags:   gormTags,
		Comment:    col.Comment,
		JSONName:   naming.ToSnakeCase(col.Name),
		IsNullable: col.IsNullable,
	}
}

// buildGormTags generates gorm struct tags for a column
func (g *Generator) buildGormTags(col *dbparser.Column) []string {
	var tags []string

	// Column name
	tags = append(tags, fmt.Sprintf("column:%s", col.Name))

	// Type hint for special types
	tags = g.appendTypeTag(tags, col)

	// Primary key & auto increment
	if col.IsPrimaryKey {
		tags = append(tags, "primaryKey")
	}
	if col.IsAutoIncr {
		tags = append(tags, "autoIncrement")
	}

	// Constraints
	if !col.IsNullable {
		tags = append(tags, "not null")
	}
	if col.IsUnique && !col.IsPrimaryKey {
		tags = append(tags, "unique")
	}

	// Default value
	tags = g.appendDefaultTag(tags, col)

	return tags
}

// appendTypeTag adds type-specific GORM tags
func (g *Generator) appendTypeTag(tags []string, col *dbparser.Column) []string {
	switch col.ColumnType {
	case "uuid":
		return append(tags, "type:uuid")
	case "jsonb":
		return append(tags, "type:jsonb")
	case "json":
		return append(tags, "type:json")
	case "text":
		return append(tags, "type:text")
	}
	return tags
}

// appendDefaultTag adds default value GORM tags
func (g *Generator) appendDefaultTag(tags []string, col *dbparser.Column) []string {
	if !col.HasDefault || col.IsAutoIncr {
		return tags
	}

	def := col.Default

	switch {
	case strings.Contains(def, "now()") || strings.Contains(def, "CURRENT_TIMESTAMP"):
		tags = append(tags, "autoCreateTime")
	case strings.Contains(def, "gen_random_uuid()") || strings.Contains(def, "uuid_generate_v4()"):
		tags = append(tags, "default:gen_random_uuid()")
	case def != "" && !strings.Contains(def, "nextval"):
		tags = append(tags, fmt.Sprintf("default:%s", def))
	}

	// Auto update time for updated_at column
	if col.Name == "updated_at" {
		tags = append(tags, "autoUpdateTime")
	}

	return tags
}

// renderModelTemplate renders the model template
func (g *Generator) renderModelTemplate(data *modelData) ([]byte, error) {
	tmpl, err := template.New("model").Parse(modelTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	// Format Go code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// Return unformatted if formatting fails (for debugging)
		return buf.Bytes(), nil
	}

	return formatted, nil
}

// generateEnums generates enum type definitions
func (g *Generator) generateEnums(dir string) error {
	data := g.buildEnumData()

	content, err := g.renderEnumTemplate(data)
	if err != nil {
		return err
	}

	filename := filepath.Join(dir, "enums_gen.go")
	g.logVerbose("  Generating enums: %s\n", filename)

	return os.WriteFile(filename, content, 0o644)
}

type enumData struct {
	PackageName string
	Enums       []enumType
}

type enumType struct {
	Name   string
	Values []enumValue
}

type enumValue struct {
	Name  string
	Value string
}

func (g *Generator) buildEnumData() *enumData {
	var enums []enumType

	for _, e := range g.schema.Enums {
		enumName := naming.ToPascalCase(e.Name)
		var values []enumValue

		for _, v := range e.Values {
			values = append(values, enumValue{
				Name:  enumName + naming.ToPascalCase(v),
				Value: v,
			})
		}

		enums = append(enums, enumType{
			Name:   enumName,
			Values: values,
		})
	}

	return &enumData{
		PackageName: g.opts.ModelPackage,
		Enums:       enums,
	}
}

func (g *Generator) renderEnumTemplate(data *enumData) ([]byte, error) {
	tmpl, err := template.New("enums").Parse(enumTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return buf.Bytes(), nil
	}

	return formatted, nil
}

// Templates

const modelTemplate = `// Code generated by ss-plugin-db. DO NOT EDIT.
// Source: ss db gorm gen

package {{.PackageName}}

{{if .Imports}}
import (
{{range .Imports}}	"{{.}}"
{{end}})
{{end}}

{{if .Comment}}// {{.StructName}} - {{.Comment}}{{else}}// {{.StructName}} represents the {{.TableName}} table{{end}}
type {{.StructName}} struct {
{{range .Fields}}	{{.Name}} {{.Type}} ` + "`" + `gorm:"{{range $i, $tag := .GormTags}}{{if $i}};{{end}}{{$tag}}{{end}}" json:"{{.JSONName}}{{if .IsNullable}},omitempty{{end}}"` + "`" + `{{if .Comment}} // {{.Comment}}{{end}}
{{end}}{{if .SoftDelete}}	DeletedAt gorm.DeletedAt ` + "`" + `gorm:"index" json:"-"` + "`" + `
{{end}}}

// TableName returns the table name for GORM
func ({{.StructName}}) TableName() string {
	return "{{.TableName}}"
}
`

const enumTemplate = `// Code generated by ss-plugin-db. DO NOT EDIT.
// Source: ss db gorm gen

package {{.PackageName}}

{{range $enum := .Enums}}
// {{$enum.Name}} represents the {{$enum.Name}} enum type
type {{$enum.Name}} string

const (
{{range $v := $enum.Values}}	{{$v.Name}} {{$enum.Name}} = "{{$v.Value}}"
{{end}})

// Valid{{$enum.Name}}Values returns all valid values for {{$enum.Name}}
func Valid{{$enum.Name}}Values() []{{$enum.Name}} {
	return []{{$enum.Name}}{
{{range $enum.Values}}		{{.Name}},
{{end}}	}
}

{{end}}
`
