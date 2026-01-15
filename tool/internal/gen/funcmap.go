package gen

import (
	"strings"
	"text/template"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/ssgohq/ssgo/internal/util/naming"
)

// titleCaser is used for title case conversion
var titleCaser = cases.Title(language.English)

// DefaultFuncMap returns the default template function map.
// All generators inherit these functions and can add their own.
func DefaultFuncMap() template.FuncMap {
	return template.FuncMap{
		// Naming conventions
		"ToSnakeCase":  naming.ToSnakeCase,
		"ToCamelCase":  naming.ToCamelCase,
		"ToPascalCase": naming.ToPascalCase,
		"ToKebabCase":  naming.ToKebabCase,

		// String operations
		"lower":      strings.ToLower,
		"upper":      strings.ToUpper,
		"title":      titleCaser.String,
		"trimPrefix": strings.TrimPrefix,
		"trimSuffix": strings.TrimSuffix,
		"contains":   strings.Contains,
		"hasPrefix":  strings.HasPrefix,
		"hasSuffix":  strings.HasSuffix,
		"replace":    strings.Replace,
		"replaceAll": strings.ReplaceAll,
		"split":      strings.Split,
		"join":       strings.Join,

		// Utility functions
		"default": defaultValue,
		"ternary": ternary,
		"add":     add,
		"sub":     sub,
	}
}

// MergeFuncMap merges additional functions into the base func map.
// Additional functions override base functions with the same name.
func MergeFuncMap(base, additional template.FuncMap) template.FuncMap {
	result := make(template.FuncMap, len(base)+len(additional))
	for k, v := range base {
		result[k] = v
	}
	for k, v := range additional {
		result[k] = v
	}
	return result
}

// defaultValue returns def if val is nil or empty string.
func defaultValue(def, val any) any {
	if val == nil {
		return def
	}
	if s, ok := val.(string); ok && s == "" {
		return def
	}
	return val
}

// ternary returns ifTrue if cond is true, otherwise ifFalse.
func ternary(cond bool, ifTrue, ifFalse any) any {
	if cond {
		return ifTrue
	}
	return ifFalse
}

// add returns a + b.
func add(a, b int) int {
	return a + b
}

// sub returns a - b.
func sub(a, b int) int {
	return a - b
}
