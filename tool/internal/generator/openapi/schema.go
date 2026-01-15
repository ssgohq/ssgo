// Package openapi provides OpenAPI 3.0 spec generation from .api files
package openapi

import (
	"strconv"
	"strings"
)

// OpenAPI 3.0 Spec types

// OpenAPISpec represents an OpenAPI 3.0 specification
type OpenAPISpec struct {
	OpenAPI    string                `json:"openapi"              yaml:"openapi"`
	Info       Info                  `json:"info"                 yaml:"info"`
	Servers    []Server              `json:"servers,omitempty"    yaml:"servers,omitempty"`
	Paths      map[string]PathItem   `json:"paths"                yaml:"paths"`
	Components *Components           `json:"components,omitempty" yaml:"components,omitempty"`
	Security   []SecurityRequirement `json:"security,omitempty"   yaml:"security,omitempty"`
	Tags       []Tag                 `json:"tags,omitempty"       yaml:"tags,omitempty"`
}

// Info represents OpenAPI info object
type Info struct {
	Title       string   `json:"title"                 yaml:"title"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Version     string   `json:"version"               yaml:"version"`
	Contact     *Contact `json:"contact,omitempty"     yaml:"contact,omitempty"`
}

// Contact represents OpenAPI contact object
type Contact struct {
	Name  string `json:"name,omitempty"  yaml:"name,omitempty"`
	Email string `json:"email,omitempty" yaml:"email,omitempty"`
	URL   string `json:"url,omitempty"   yaml:"url,omitempty"`
}

// Server represents OpenAPI server object
type Server struct {
	URL         string `json:"url"                   yaml:"url"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// PathItem represents OpenAPI path item object
type PathItem struct {
	Get     *Operation `json:"get,omitempty"     yaml:"get,omitempty"`
	Post    *Operation `json:"post,omitempty"    yaml:"post,omitempty"`
	Put     *Operation `json:"put,omitempty"     yaml:"put,omitempty"`
	Delete  *Operation `json:"delete,omitempty"  yaml:"delete,omitempty"`
	Patch   *Operation `json:"patch,omitempty"   yaml:"patch,omitempty"`
	Head    *Operation `json:"head,omitempty"    yaml:"head,omitempty"`
	Options *Operation `json:"options,omitempty" yaml:"options,omitempty"`
}

// Operation represents OpenAPI operation object
type Operation struct {
	Tags        []string              `json:"tags,omitempty"        yaml:"tags,omitempty"`
	Summary     string                `json:"summary,omitempty"     yaml:"summary,omitempty"`
	Description string                `json:"description,omitempty" yaml:"description,omitempty"`
	OperationID string                `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	Parameters  []Parameter           `json:"parameters,omitempty"  yaml:"parameters,omitempty"`
	RequestBody *RequestBody          `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	Responses   map[string]Response   `json:"responses"             yaml:"responses"`
	Security    []SecurityRequirement `json:"security,omitempty"    yaml:"security,omitempty"`
	Deprecated  bool                  `json:"deprecated,omitempty"  yaml:"deprecated,omitempty"`
}

// Parameter represents OpenAPI parameter object
type Parameter struct {
	Name        string  `json:"name"                  yaml:"name"`
	In          string  `json:"in"                    yaml:"in"` // path, query, header, cookie
	Description string  `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool    `json:"required"              yaml:"required"`
	Schema      *Schema `json:"schema"                yaml:"schema"`
	Example     any     `json:"example,omitempty"     yaml:"example,omitempty"`
}

// RequestBody represents OpenAPI request body object
type RequestBody struct {
	Description string               `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool                 `json:"required,omitempty"    yaml:"required,omitempty"`
	Content     map[string]MediaType `json:"content"               yaml:"content"`
}

// MediaType represents OpenAPI media type object
type MediaType struct {
	Schema  *Schema `json:"schema"            yaml:"schema"`
	Example any     `json:"example,omitempty" yaml:"example,omitempty"`
}

// Response represents OpenAPI response object
type Response struct {
	Description string               `json:"description"       yaml:"description"`
	Content     map[string]MediaType `json:"content,omitempty" yaml:"content,omitempty"`
	Headers     map[string]Header    `json:"headers,omitempty" yaml:"headers,omitempty"`
}

// Header represents OpenAPI header object
type Header struct {
	Description string  `json:"description,omitempty" yaml:"description,omitempty"`
	Schema      *Schema `json:"schema,omitempty"      yaml:"schema,omitempty"`
}

// Schema represents OpenAPI/JSON Schema object
type Schema struct {
	Type                 string             `json:"type,omitempty"                 yaml:"type,omitempty"`
	Format               string             `json:"format,omitempty"               yaml:"format,omitempty"`
	Title                string             `json:"title,omitempty"                yaml:"title,omitempty"`
	Description          string             `json:"description,omitempty"          yaml:"description,omitempty"`
	Properties           map[string]*Schema `json:"properties,omitempty"           yaml:"properties,omitempty"`
	Items                *Schema            `json:"items,omitempty"                yaml:"items,omitempty"`
	Required             []string           `json:"required,omitempty"             yaml:"required,omitempty"`
	Ref                  string             `json:"$ref,omitempty"                 yaml:"$ref,omitempty"`
	AdditionalProperties *Schema            `json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty"`
	MinLength            *int               `json:"minLength,omitempty"            yaml:"minLength,omitempty"`
	MaxLength            *int               `json:"maxLength,omitempty"            yaml:"maxLength,omitempty"`
	Pattern              string             `json:"pattern,omitempty"              yaml:"pattern,omitempty"`
	Minimum              *int               `json:"minimum,omitempty"              yaml:"minimum,omitempty"`
	Maximum              *int               `json:"maximum,omitempty"              yaml:"maximum,omitempty"`
	Enum                 []any              `json:"enum,omitempty"                 yaml:"enum,omitempty"`
	Default              any                `json:"default,omitempty"              yaml:"default,omitempty"`
	Example              any                `json:"example,omitempty"              yaml:"example,omitempty"`
	Nullable             bool               `json:"nullable,omitempty"             yaml:"nullable,omitempty"`
}

// Components represents OpenAPI components object
type Components struct {
	Schemas         map[string]*Schema         `json:"schemas,omitempty"         yaml:"schemas,omitempty"`
	SecuritySchemes map[string]*SecurityScheme `json:"securitySchemes,omitempty" yaml:"securitySchemes,omitempty"`
	Parameters      map[string]*Parameter      `json:"parameters,omitempty"      yaml:"parameters,omitempty"`
	RequestBodies   map[string]*RequestBody    `json:"requestBodies,omitempty"   yaml:"requestBodies,omitempty"`
	Responses       map[string]*Response       `json:"responses,omitempty"       yaml:"responses,omitempty"`
}

// SecurityScheme represents OpenAPI security scheme object
type SecurityScheme struct {
	Type             string `json:"type"                       yaml:"type"`
	Scheme           string `json:"scheme,omitempty"           yaml:"scheme,omitempty"`
	BearerFormat     string `json:"bearerFormat,omitempty"     yaml:"bearerFormat,omitempty"`
	Description      string `json:"description,omitempty"      yaml:"description,omitempty"`
	Name             string `json:"name,omitempty"             yaml:"name,omitempty"`
	In               string `json:"in,omitempty"               yaml:"in,omitempty"`
	OpenIDConnectURL string `json:"openIdConnectUrl,omitempty" yaml:"openIdConnectUrl,omitempty"`
}

// SecurityRequirement represents OpenAPI security requirement object
type SecurityRequirement map[string][]string

// Tag represents OpenAPI tag object
type Tag struct {
	Name        string `json:"name"                  yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// GoTypeToOpenAPI converts a Go type to OpenAPI type and format
func GoTypeToOpenAPI(goType string) (schemaType, format string) {
	goType = strings.TrimPrefix(goType, "*")

	switch goType {
	case "string":
		return "string", ""
	case "int":
		return "integer", "int32"
	case "int8":
		return "integer", "int32"
	case "int16":
		return "integer", "int32"
	case "int32":
		return "integer", "int32"
	case "int64":
		return "integer", "int64"
	case "uint":
		return "integer", "int32"
	case "uint8":
		return "integer", "int32"
	case "uint16":
		return "integer", "int32"
	case "uint32":
		return "integer", "int32"
	case "uint64":
		return "integer", "int64"
	case "float32":
		return "number", "float"
	case "float64":
		return "number", "double"
	case "bool":
		return "boolean", ""
	case "byte":
		return "string", "byte"
	case "time.Time":
		return "string", "date-time"
	case "interface{}", "any":
		return "object", ""
	default:
		if strings.HasPrefix(goType, "[]") {
			return "array", ""
		}
		if strings.HasPrefix(goType, "map[") {
			return "object", ""
		}
		return "object", ""
	}
}

// IsBasicType returns true if the type is a basic Go type
func IsBasicType(typeName string) bool {
	typeName = strings.TrimPrefix(typeName, "*")
	switch typeName {
	case "bool",
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64",
		"complex64", "complex128",
		"string", "byte", "rune",
		"interface{}", "any",
		"time.Time":
		return true
	}
	return false
}

// ValidateConstraints holds validation constraints parsed from validate tags
type ValidateConstraints struct {
	Required  bool
	MinLength *int
	MaxLength *int
	Min       *int
	Max       *int
	Pattern   string
	Email     bool
	URL       bool
	Enum      []string
}

// ParseValidateTag parses a validate tag and returns constraints
func ParseValidateTag(tag string) *ValidateConstraints {
	if tag == "" {
		return nil
	}

	constraints := &ValidateConstraints{}
	parts := strings.Split(tag, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if idx := strings.Index(part, "="); idx > 0 {
			key := part[:idx]
			value := part[idx+1:]

			switch key {
			case "min":
				if v, err := strconv.Atoi(value); err == nil {
					constraints.Min = &v
					constraints.MinLength = &v
				}
			case "max":
				if v, err := strconv.Atoi(value); err == nil {
					constraints.Max = &v
					constraints.MaxLength = &v
				}
			case "len":
				if v, err := strconv.Atoi(value); err == nil {
					constraints.MinLength = &v
					constraints.MaxLength = &v
				}
			case "oneof":
				constraints.Enum = strings.Fields(value)
			}
		} else {
			switch part {
			case "required":
				constraints.Required = true
			case "email":
				constraints.Email = true
			case "url", "uri":
				constraints.URL = true
			}
		}
	}

	return constraints
}

// ApplyConstraintsToSchema applies validation constraints to a schema
func ApplyConstraintsToSchema(schema *Schema, constraints *ValidateConstraints, schemaType string) {
	if constraints == nil {
		return
	}

	switch schemaType {
	case "string":
		schema.MinLength = constraints.MinLength
		schema.MaxLength = constraints.MaxLength
		if constraints.Email {
			schema.Format = "email"
		}
		if constraints.URL {
			schema.Format = "uri"
		}
		if constraints.Pattern != "" {
			schema.Pattern = constraints.Pattern
		}
	case "integer", "number":
		schema.Minimum = constraints.Min
		schema.Maximum = constraints.Max
	}

	if len(constraints.Enum) > 0 {
		for _, v := range constraints.Enum {
			schema.Enum = append(schema.Enum, v)
		}
	}
}

// ParseOptionsTag parses options from tags like "options=active|inactive"
func ParseOptionsTag(tag string) []string {
	if !strings.Contains(tag, "options=") {
		return nil
	}

	parts := strings.Split(tag, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "options=") {
			optionsStr := strings.TrimPrefix(part, "options=")
			return strings.Split(optionsStr, "|")
		}
	}
	return nil
}

// ParseDefaultValue parses default value from tags
func ParseDefaultValue(tag string) string {
	parts := strings.Split(tag, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "default=") {
			return strings.TrimPrefix(part, "default=")
		}
	}
	return ""
}

// SchemaRef creates a JSON schema reference
func SchemaRef(typeName string) *Schema {
	return &Schema{
		Ref: "#/components/schemas/" + typeName,
	}
}

// ArraySchema creates an array schema
func ArraySchema(items *Schema) *Schema {
	return &Schema{
		Type:  "array",
		Items: items,
	}
}

// ObjectSchema creates an object schema
func ObjectSchema() *Schema {
	return &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
	}
}
