package mysql

import (
	"strings"

	"github.com/ssgohq/ssgo/internal/dbparser"
)

// Mapper implements dbparser.TypeMapper for MySQL
type Mapper struct{}

// NewMapper creates a new MySQL type mapper
func NewMapper() dbparser.TypeMapper {
	return &Mapper{}
}

// MapToGo maps a MySQL column to Go type
func (m *Mapper) MapToGo(col *dbparser.Column, opts dbparser.MapperOptions) (goType, importPath string) {
	dataType := strings.ToLower(col.DataType)

	// Special case: tinyint(1) is boolean
	if dataType == "tinyint" {
		if strings.Contains(strings.ToLower(col.ColumnType), "tinyint(1)") {
			return dbparser.ApplyNullable("bool", col, opts), ""
		}
	}

	// Special case: bit(1) is boolean
	if dataType == "bit" {
		if strings.Contains(strings.ToLower(col.ColumnType), "bit(1)") {
			return dbparser.ApplyNullable("bool", col, opts), ""
		}
	}

	info, ok := mysqlTypeMap[dataType]
	if !ok {
		return "interface{}", ""
	}

	goType = info.GoType
	importPath = info.ImportPath

	// Handle unsigned types
	if col.IsUnsigned {
		if unsigned, ok := unsignedTypeMap[goType]; ok {
			goType = unsigned
		}
	}

	// Handle JSON option
	if dataType == "json" && !opts.JSONAsRawMessage {
		return dbparser.ApplyNullable("string", col, opts), ""
	}

	// Handle decimal package option
	if dataType == "decimal" || dataType == "numeric" {
		if opts.DecimalPackage == "" {
			goType = "float64"
			importPath = ""
		}
	}

	// Handle time as string option
	if dataType == "time" && opts.TimeAsString {
		return dbparser.ApplyNullable("string", col, opts), ""
	}

	return dbparser.ApplyNullable(goType, col, opts), importPath
}

// typeInfo holds Go type information
type typeInfo struct {
	GoType     string
	ImportPath string
}

// MySQL to Go type mapping
var mysqlTypeMap = map[string]typeInfo{
	// Integer types
	"tinyint":   {GoType: "int8"},
	"smallint":  {GoType: "int16"},
	"mediumint": {GoType: "int32"},
	"int":       {GoType: "int32"},
	"integer":   {GoType: "int32"},
	"bigint":    {GoType: "int64"},

	// Floating point
	"float":   {GoType: "float32"},
	"double":  {GoType: "float64"},
	"decimal": {GoType: "decimal.Decimal", ImportPath: "github.com/shopspring/decimal"},
	"numeric": {GoType: "decimal.Decimal", ImportPath: "github.com/shopspring/decimal"},

	// Boolean
	"bit":     {GoType: "[]byte"},
	"bool":    {GoType: "bool"},
	"boolean": {GoType: "bool"},

	// String types
	"char":       {GoType: "string"},
	"varchar":    {GoType: "string"},
	"tinytext":   {GoType: "string"},
	"text":       {GoType: "string"},
	"mediumtext": {GoType: "string"},
	"longtext":   {GoType: "string"},
	"enum":       {GoType: "string"},
	"set":        {GoType: "string"},

	// Binary
	"binary":     {GoType: "[]byte"},
	"varbinary":  {GoType: "[]byte"},
	"tinyblob":   {GoType: "[]byte"},
	"blob":       {GoType: "[]byte"},
	"mediumblob": {GoType: "[]byte"},
	"longblob":   {GoType: "[]byte"},

	// Date/Time
	"date":      {GoType: "time.Time", ImportPath: "time"},
	"datetime":  {GoType: "time.Time", ImportPath: "time"},
	"timestamp": {GoType: "time.Time", ImportPath: "time"},
	"time":      {GoType: "string"},
	"year":      {GoType: "int16"},

	// JSON
	"json": {GoType: "json.RawMessage", ImportPath: "encoding/json"},

	// Spatial types (as string for simplicity)
	"geometry":           {GoType: "string"},
	"point":              {GoType: "string"},
	"linestring":         {GoType: "string"},
	"polygon":            {GoType: "string"},
	"multipoint":         {GoType: "string"},
	"multilinestring":    {GoType: "string"},
	"multipolygon":       {GoType: "string"},
	"geometrycollection": {GoType: "string"},
}

// Unsigned type mapping
var unsignedTypeMap = map[string]string{
	"int8":  "uint8",
	"int16": "uint16",
	"int32": "uint32",
	"int64": "uint64",
}
