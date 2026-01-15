package hertz

import (
	"strings"

	"github.com/ssgohq/ssgo/tool/internal/spec/api"
)

// TypeData represents template data for types.go
type TypeData struct {
	Types []TypeDefData
}

// TypeDefData represents a type definition for template
type TypeDefData struct {
	Name    string
	Members []MemberData
	IsAlias bool   // true if this is a type alias
	AliasOf string // the aliased type name (only set if IsAlias is true)
}

// MemberData represents a struct member for template
type MemberData struct {
	Name         string
	Type         string
	Tag          string
	Comment      string
	JSONTag      string   // json tag value for validation
	Validate     string   // validate tag value
	Required     bool     // true if field is required
	Optional     bool     // true if field is optional
	Default      string   // default value string
	DefaultValue string   // default value for code generation (with proper quotes)
	ZeroValue    string   // zero value for the type
	Range        string   // range constraint e.g., "1:100"
	RangeMin     *int64   // minimum value
	RangeMax     *int64   // maximum value
	Options      []string // allowed values
}

// getJSONTagValue extracts the JSON tag name from member spec
func getJSONTagValue(m spec.MemberSpec) string {
	if m.JsonTag != "" {
		// Remove modifiers like ,omitempty
		if idx := strings.Index(m.JsonTag, ","); idx != -1 {
			return m.JsonTag[:idx]
		}
		return m.JsonTag
	}
	if m.Name != "" {
		return toJSONFieldName(m.Name)
	}
	return ""
}

// getZeroValue returns the zero value for a Go type
func getZeroValue(goType string) string {
	switch {
	case goType == "string":
		return `""`
	case goType == "bool":
		return "false"
	case strings.HasPrefix(goType, "int"), strings.HasPrefix(goType, "uint"),
		goType == "float32", goType == "float64", goType == "byte", goType == "rune":
		return "0"
	case strings.HasPrefix(goType, "[]"):
		return "nil"
	case strings.HasPrefix(goType, "map"):
		return "nil"
	case strings.HasPrefix(goType, "*"):
		return "nil"
	default:
		// For custom types, use empty struct literal
		return goType + "{}"
	}
}

// getDefaultValue formats the default value for code generation
func getDefaultValue(defaultVal, goType string) string {
	if defaultVal == "" {
		return ""
	}
	// For string types, add quotes
	if goType == "string" {
		return `"` + defaultVal + `"`
	}
	// For other types, use as-is
	return defaultVal
}

// toJSONFieldName converts a Go field name to JSON field name (camelCase to snake_case or keep camelCase)
func toJSONFieldName(name string) string {
	// Simple conversion: keep as lowercase first letter (camelCase)
	if len(name) == 0 {
		return name
	}
	return strings.ToLower(name[:1]) + name[1:]
}
