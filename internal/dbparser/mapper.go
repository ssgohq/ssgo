package dbparser

import "strings"

// TypeMapper maps SQL types to Go types
type TypeMapper interface {
	// MapToGo maps a SQL column to Go type
	// Returns: goType, importPath (empty if builtin)
	MapToGo(col *Column, opts MapperOptions) (goType, importPath string)
}

// MapperOptions configures type mapping behavior
type MapperOptions struct {
	// NullableAsPointer: use *string instead of sql.NullString (default: true)
	NullableAsPointer bool

	// UseNullTypes: use sql.Null* types for nullable columns (when NullableAsPointer=false)
	UseNullTypes bool

	// JSONAsRawMessage: use json.RawMessage instead of string for json/jsonb (default: true)
	JSONAsRawMessage bool

	// UUIDPackage: package for UUID type
	// Options: "google" (github.com/google/uuid), "satori" (github.com/satori/go.uuid)
	UUIDPackage string

	// DecimalPackage: package for decimal type
	// Options: "shopspring" (github.com/shopspring/decimal), "" (use float64)
	DecimalPackage string

	// TimeAsString: use string for time columns instead of time.Time
	TimeAsString bool
}

// DefaultMapperOptions returns default mapping options
func DefaultMapperOptions() MapperOptions {
	return MapperOptions{
		NullableAsPointer: true,
		UseNullTypes:      false,
		JSONAsRawMessage:  true,
		UUIDPackage:       "google",
		DecimalPackage:    "shopspring",
		TimeAsString:      false,
	}
}

// ApplyNullable wraps the Go type with pointer or sql.Null* based on options
func ApplyNullable(goType string, col *Column, opts MapperOptions) string {
	// Primary key is never nullable in Go
	if col.IsPrimaryKey {
		return goType
	}

	// Not nullable in DB
	if !col.IsNullable {
		return goType
	}

	// Slice/map types are already nullable
	if strings.HasPrefix(goType, "[]") || strings.HasPrefix(goType, "map[") {
		return goType
	}

	// Use pointer type (default)
	if opts.NullableAsPointer {
		return "*" + goType
	}

	// Use sql.Null* types
	if opts.UseNullTypes {
		return mapToNullType(goType)
	}

	return "*" + goType
}

func mapToNullType(goType string) string {
	switch goType {
	case "int16":
		return "sql.NullInt16"
	case "int32":
		return "sql.NullInt32"
	case "int64":
		return "sql.NullInt64"
	case "float32":
		return "sql.NullFloat64"
	case "float64":
		return "sql.NullFloat64"
	case "bool":
		return "sql.NullBool"
	case "string":
		return "sql.NullString"
	case "time.Time":
		return "sql.NullTime"
	default:
		return "*" + goType
	}
}

// TypeInfo holds Go type information
type TypeInfo struct {
	GoType     string
	ImportPath string
}

// Package-level mapper constructors
var (
	newPostgresMapper func() TypeMapper
	newMySQLMapper    func() TypeMapper
)

// RegisterPostgresMapper registers the PostgreSQL mapper constructor
func RegisterPostgresMapper(constructor func() TypeMapper) {
	newPostgresMapper = constructor
}

// RegisterMySQLMapper registers the MySQL mapper constructor
func RegisterMySQLMapper(constructor func() TypeMapper) {
	newMySQLMapper = constructor
}

// NewTypeMapper creates a type mapper for the given database type
func NewTypeMapper(dbType DatabaseType) TypeMapper {
	switch dbType {
	case DatabaseTypePostgres:
		if newPostgresMapper != nil {
			return newPostgresMapper()
		}
	case DatabaseTypeMySQL:
		if newMySQLMapper != nil {
			return newMySQLMapper()
		}
	}
	return nil
}

// MapColumnsToGo maps all columns in a table to Go types
func MapColumnsToGo(table *Table, mapper TypeMapper, opts MapperOptions) {
	for _, col := range table.Columns {
		col.GoType, col.ImportPath = mapper.MapToGo(col, opts)
	}
}

// GetRequiredImports returns all required imports for a table's columns
func GetRequiredImports(table *Table) []string {
	importSet := make(map[string]bool)

	for _, col := range table.Columns {
		if col.ImportPath != "" {
			importSet[col.ImportPath] = true
		}
	}

	var imports []string
	for imp := range importSet {
		imports = append(imports, imp)
	}
	return imports
}
