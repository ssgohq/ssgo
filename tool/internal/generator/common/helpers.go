// Package common provides shared utilities for code generators
package common

import (
	"bytes"
	"strings"
	"unicode"
)

// EntityInfo represents information about an entity
type EntityInfo struct {
	Name       string // PascalCase name (e.g., User)
	SnakeName  string // snake_case name (e.g., user)
	CamelName  string // camelCase name (e.g., user)
	PluralName string // Plural form (e.g., Users)
}

// NewEntityInfo creates a new EntityInfo from entity name
func NewEntityInfo(name string) EntityInfo {
	pascalName := ToPascalCase(name)
	return EntityInfo{
		Name:       pascalName,
		SnakeName:  ToSnakeCase(name),
		CamelName:  ToCamelCase(name),
		PluralName: Pluralize(pascalName),
	}
}

// MethodInfo represents information about a service method
type MethodInfo struct {
	Name     string // Method name (e.g., GetUser)
	Request  string // Request type (e.g., GetUserRequest)
	Response string // Response type (e.g., GetUserResponse)
}

// QueryInfo represents information about a SQLC query
type QueryInfo struct {
	Name       string  // Query function name (e.g., GetUserByID)
	ReturnType string  // Return type (e.g., *User, []User)
	Params     []Param // Function parameters
	IsMany     bool    // Returns multiple rows
	IsExec     bool    // Exec without return
	ModelName  string  // Associated model name (e.g., User)
}

// Param represents a function parameter
type Param struct {
	Name string
	Type string
}

// ToSnakeCase converts a string to snake_case
func ToSnakeCase(s string) string {
	var result bytes.Buffer
	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			// Check if previous char is lowercase or next char is lowercase
			if i+1 < len(s) && unicode.IsLower(rune(s[i+1])) {
				result.WriteByte('_')
			} else if unicode.IsLower(rune(s[i-1])) {
				result.WriteByte('_')
			}
		}
		result.WriteRune(unicode.ToLower(r))
	}
	return result.String()
}

// ToPascalCase converts a string to PascalCase
func ToPascalCase(s string) string {
	// Handle snake_case input
	if strings.Contains(s, "_") {
		parts := strings.Split(s, "_")
		var result bytes.Buffer
		for _, part := range parts {
			if len(part) > 0 {
				result.WriteString(strings.ToUpper(string(part[0])))
				result.WriteString(strings.ToLower(part[1:]))
			}
		}
		return result.String()
	}

	// Handle kebab-case input
	if strings.Contains(s, "-") {
		parts := strings.Split(s, "-")
		var result bytes.Buffer
		for _, part := range parts {
			if len(part) > 0 {
				result.WriteString(strings.ToUpper(string(part[0])))
				result.WriteString(strings.ToLower(part[1:]))
			}
		}
		return result.String()
	}

	// Already PascalCase or camelCase
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(string(s[0])) + s[1:]
}

// ToCamelCase converts a string to camelCase
func ToCamelCase(s string) string {
	pascal := ToPascalCase(s)
	if len(pascal) == 0 {
		return pascal
	}
	return strings.ToLower(string(pascal[0])) + pascal[1:]
}

// ToKebabCase converts a string to kebab-case
func ToKebabCase(s string) string {
	snake := ToSnakeCase(s)
	return strings.ReplaceAll(snake, "_", "-")
}

// Pluralize adds 's' to the end of a word (simple pluralization)
func Pluralize(s string) string {
	if s == "" {
		return s
	}
	lastChar := s[len(s)-1]
	switch lastChar {
	case 's', 'x', 'z':
		return s + "es"
	case 'y':
		if len(s) > 1 {
			prevChar := s[len(s)-2]
			if prevChar != 'a' && prevChar != 'e' && prevChar != 'i' && prevChar != 'o' && prevChar != 'u' {
				return s[:len(s)-1] + "ies"
			}
		}
		return s + "s"
	default:
		return s + "s"
	}
}

// HandlerFileName generates handler file name from service name
func HandlerFileName(serviceName string) string {
	return ToSnakeCase(serviceName) + "_handler.go"
}

// RepositoryFileName generates repository file name from entity name
func RepositoryFileName(entityName string) string {
	return ToSnakeCase(entityName) + "_repository.go"
}

// ModelFileName generates model file name from entity name
func ModelFileName(entityName string) string {
	return ToSnakeCase(entityName) + ".go"
}

// ReadModuleFromGoMod reads the module name from go.mod content
func ReadModuleFromGoMod(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimPrefix(line, "module ")
		}
	}
	return ""
}
