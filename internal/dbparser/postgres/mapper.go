package postgres

import (
	"strings"

	"github.com/ssgohq/ssgo/internal/dbparser"
)

// Mapper implements dbparser.TypeMapper for PostgreSQL
type Mapper struct{}

// NewMapper creates a new PostgreSQL type mapper
func NewMapper() dbparser.TypeMapper {
	return &Mapper{}
}

// MapToGo maps a PostgreSQL column to Go type
func (m *Mapper) MapToGo(col *dbparser.Column, opts dbparser.MapperOptions) (goType, importPath string) {
	// Handle array types
	if col.IsArray {
		return m.mapArrayType(col, opts)
	}

	// Handle enum types
	if col.EnumName != "" {
		goType = "string"
		return dbparser.ApplyNullable(goType, col, opts), ""
	}

	// Lookup type
	dataType := strings.ToLower(col.DataType)
	info, ok := pgTypeMap[dataType]
	if !ok {
		// Fallback
		return "interface{}", ""
	}

	goType = info.GoType
	importPath = info.ImportPath

	// Handle JSON option
	if (dataType == "json" || dataType == "jsonb") && !opts.JSONAsRawMessage {
		return dbparser.ApplyNullable("string", col, opts), ""
	}

	// Handle UUID package option
	if dataType == "uuid" {
		switch opts.UUIDPackage {
		case "satori":
			importPath = "github.com/satori/go.uuid"
		default:
			importPath = "github.com/google/uuid"
		}
	}

	// Handle decimal package option
	if dataType == "numeric" || dataType == "decimal" {
		if opts.DecimalPackage == "" {
			goType = "float64"
			importPath = ""
		} else {
			importPath = "github.com/shopspring/decimal"
		}
	}

	return dbparser.ApplyNullable(goType, col, opts), importPath
}

func (m *Mapper) mapArrayType(col *dbparser.Column, opts dbparser.MapperOptions) (string, string) {
	// Get base type
	baseType := strings.ToLower(col.DataType)

	if info, ok := pgArrayTypeMap[baseType]; ok {
		return info.GoType, info.ImportPath
	}

	// Try to map base type and make it a slice
	if info, ok := pgTypeMap[baseType]; ok {
		return "[]" + info.GoType, info.ImportPath
	}

	// Default array to []interface{}
	return "[]interface{}", ""
}

// typeInfo holds Go type information
type typeInfo struct {
	GoType     string
	ImportPath string
}

// PostgreSQL to Go type mapping
var pgTypeMap = map[string]typeInfo{
	// Integer types
	"int2":        {GoType: "int16"},
	"int4":        {GoType: "int32"},
	"int8":        {GoType: "int64"},
	"smallint":    {GoType: "int16"},
	"integer":     {GoType: "int32"},
	"bigint":      {GoType: "int64"},
	"smallserial": {GoType: "int16"},
	"serial":      {GoType: "int32"},
	"bigserial":   {GoType: "int64"},
	"serial4":     {GoType: "int32"},
	"serial8":     {GoType: "int64"},

	// Floating point
	"float4":           {GoType: "float32"},
	"float8":           {GoType: "float64"},
	"real":             {GoType: "float32"},
	"double precision": {GoType: "float64"},
	"numeric":          {GoType: "decimal.Decimal", ImportPath: "github.com/shopspring/decimal"},
	"decimal":          {GoType: "decimal.Decimal", ImportPath: "github.com/shopspring/decimal"},
	"money":            {GoType: "string"},

	// Boolean
	"bool":    {GoType: "bool"},
	"boolean": {GoType: "bool"},

	// String types
	"text":              {GoType: "string"},
	"varchar":           {GoType: "string"},
	"character varying": {GoType: "string"},
	"char":              {GoType: "string"},
	"character":         {GoType: "string"},
	"bpchar":            {GoType: "string"},
	"name":              {GoType: "string"},
	"citext":            {GoType: "string"},

	// Binary
	"bytea": {GoType: "[]byte"},

	// Date/Time
	"date":                        {GoType: "time.Time", ImportPath: "time"},
	"time":                        {GoType: "time.Time", ImportPath: "time"},
	"timetz":                      {GoType: "time.Time", ImportPath: "time"},
	"time with time zone":         {GoType: "time.Time", ImportPath: "time"},
	"time without time zone":      {GoType: "time.Time", ImportPath: "time"},
	"timestamp":                   {GoType: "time.Time", ImportPath: "time"},
	"timestamptz":                 {GoType: "time.Time", ImportPath: "time"},
	"timestamp with time zone":    {GoType: "time.Time", ImportPath: "time"},
	"timestamp without time zone": {GoType: "time.Time", ImportPath: "time"},
	"interval":                    {GoType: "string"},

	// JSON
	"json":  {GoType: "json.RawMessage", ImportPath: "encoding/json"},
	"jsonb": {GoType: "json.RawMessage", ImportPath: "encoding/json"},

	// UUID
	"uuid": {GoType: "uuid.UUID", ImportPath: "github.com/google/uuid"},

	// Network
	"inet":     {GoType: "string"},
	"cidr":     {GoType: "string"},
	"macaddr":  {GoType: "string"},
	"macaddr8": {GoType: "string"},

	// Geometric (as string for simplicity)
	"point":   {GoType: "string"},
	"line":    {GoType: "string"},
	"lseg":    {GoType: "string"},
	"box":     {GoType: "string"},
	"path":    {GoType: "string"},
	"polygon": {GoType: "string"},
	"circle":  {GoType: "string"},

	// Range types
	"int4range":      {GoType: "string"},
	"int8range":      {GoType: "string"},
	"numrange":       {GoType: "string"},
	"tsrange":        {GoType: "string"},
	"tstzrange":      {GoType: "string"},
	"daterange":      {GoType: "string"},
	"int4multirange": {GoType: "string"},
	"int8multirange": {GoType: "string"},
	"nummultirange":  {GoType: "string"},
	"tsmultirange":   {GoType: "string"},
	"tstzmultirange": {GoType: "string"},
	"datemultirange": {GoType: "string"},

	// Other
	"xml":         {GoType: "string"},
	"bit":         {GoType: "string"},
	"varbit":      {GoType: "string"},
	"bit varying": {GoType: "string"},
	"tsvector":    {GoType: "string"},
	"tsquery":     {GoType: "string"},
	"oid":         {GoType: "uint32"},
}

// PostgreSQL array type mapping
var pgArrayTypeMap = map[string]typeInfo{
	"bool":    {GoType: "[]bool"},
	"boolean": {GoType: "[]bool"},
	"int2":    {GoType: "[]int16"},
	"int4":    {GoType: "[]int32"},
	"int8":    {GoType: "[]int64"},
	"float4":  {GoType: "[]float32"},
	"float8":  {GoType: "[]float64"},
	"text":    {GoType: "[]string"},
	"varchar": {GoType: "[]string"},
	"uuid":    {GoType: "[]uuid.UUID", ImportPath: "github.com/google/uuid"},
}
