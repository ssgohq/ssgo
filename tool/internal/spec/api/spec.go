// Package spec provides high-level spec types for code generation
package spec

import (
	"regexp"
	"strconv"
	"strings"

	ast "github.com/ssgohq/ssgo/internal/ast/api"
)

// ServiceSpec represents a parsed service specification for code generation
type ServiceSpec struct {
	Name    string
	Module  string
	Package string
	Groups  []Group
	Types   []TypeSpec
}

// Group represents a group of routes with common configuration
type Group struct {
	Annotation *Annotation
	Routes     []RouteSpec
}

// Annotation represents @server annotation configuration
type Annotation struct {
	Prefix     string
	Group      string
	JWT        string
	Middleware []string
	Timeout    string
	MaxBytes   string
	Properties map[string]string
}

// RouteSpec represents a route specification
type RouteSpec struct {
	Method       string
	Path         string
	Handler      string
	Doc          *DocSpec
	RequestType  *TypeSpec
	ResponseType *TypeSpec
}

// DocSpec represents route documentation
type DocSpec struct {
	Summary     string
	Description string
}

// TypeSpec represents a type specification
type TypeSpec struct {
	Name    string
	Members []MemberSpec
	IsAlias bool   // true if this is a type alias
	AliasOf string // the aliased type name (only set if IsAlias is true)
}

// MemberSpec represents a struct member specification
type MemberSpec struct {
	Name      string
	Type      string
	RawType   string // The raw type expression (e.g., []int64, *User)
	JsonTag   string
	PathTag   string
	QueryTag  string
	FormTag   string
	HeaderTag string
	Validate  string
	Optional  bool
	Default   string
	IsInline  bool
	Comment   string
	Docs      []string
	// Validation rules extracted from validate tag
	Range    string   // e.g., "1:100", "1:", ":100"
	RangeMin *int64   // min value for range validation
	RangeMax *int64   // max value for range validation
	Options  []string // e.g., ["pending", "active", "done"]
}

// ParseTag parses a struct tag string and returns individual tags
func ParseTag(tag string) map[string]string {
	result := make(map[string]string)
	if tag == "" {
		return result
	}

	// Parse struct tag format: `json:"name" validate:"required"`
	parts := strings.Fields(tag)
	for _, part := range parts {
		idx := strings.Index(part, ":")
		if idx > 0 {
			key := part[:idx]
			value := strings.Trim(part[idx+1:], "\"")
			result[key] = value
		}
	}

	return result
}

// parseValidationRules extracts range and options from validate tag
func parseValidationRules(validateTag string) (rangeStr string, rangeMin, rangeMax *int64, options []string) {
	if validateTag == "" {
		return rangeStr, rangeMin, rangeMax, options
	}

	// Extract range=[min:max] pattern
	rangeRegex := regexp.MustCompile(`range=\[([^\]]*)\]`)
	if matches := rangeRegex.FindStringSubmatch(validateTag); len(matches) > 1 {
		rangeStr = matches[1]
		rangeParts := strings.Split(rangeStr, ":")
		if len(rangeParts) == 2 {
			if rangeParts[0] != "" {
				if min, err := strconv.ParseInt(rangeParts[0], 10, 64); err == nil {
					rangeMin = &min
				}
			}
			if rangeParts[1] != "" {
				if max, err := strconv.ParseInt(rangeParts[1], 10, 64); err == nil {
					rangeMax = &max
				}
			}
		}
	}

	// Extract options=a|b|c pattern
	optionsRegex := regexp.MustCompile(`options=([^,\s\]]+)`)
	if matches := optionsRegex.FindStringSubmatch(validateTag); len(matches) > 1 {
		optStr := matches[1]
		options = strings.Split(optStr, "|")
		// Trim whitespace from each option
		for i := range options {
			options[i] = strings.TrimSpace(options[i])
		}
	}

	return rangeStr, rangeMin, rangeMax, options
}

// FromAST converts an API AST to a ServiceSpec
func FromAST(apiSpec *ast.ApiSpec) (*ServiceSpec, error) {
	spec := &ServiceSpec{
		Name: "",
	}

	// Convert types
	for _, t := range apiSpec.Types {
		typeSpec := TypeSpec{
			Name:    t.Name,
			IsAlias: t.IsAlias,
		}

		// Handle type alias
		if t.IsAlias && t.AliasOf != nil {
			typeSpec.AliasOf = t.AliasOf.TypeName()
		} else {
			// Regular struct type
			for _, m := range t.Members {
				memberSpec := convertMember(m)
				typeSpec.Members = append(typeSpec.Members, memberSpec)
			}
		}

		spec.Types = append(spec.Types, typeSpec)
	}

	// Convert services
	typeMap := make(map[string]*TypeSpec)
	for i := range spec.Types {
		typeMap[spec.Types[i].Name] = &spec.Types[i]
	}

	for _, svc := range apiSpec.Services {
		if spec.Name == "" {
			spec.Name = svc.Name
		}

		group := Group{}

		if svc.Annotation != nil {
			group.Annotation = &Annotation{
				Prefix:     svc.Annotation.Prefix,
				Group:      svc.Annotation.Group,
				JWT:        svc.Annotation.JWT,
				Middleware: svc.Annotation.Middleware,
				Timeout:    svc.Annotation.Timeout,
				MaxBytes:   svc.Annotation.MaxBytes,
				Properties: svc.Annotation.Properties,
			}
		}

		for _, r := range svc.Routes {
			route := RouteSpec{
				Method:  r.Method,
				Path:    r.Path,
				Handler: r.Handler,
			}

			if r.Doc != nil {
				route.Doc = &DocSpec{
					Summary:     r.Doc.Summary,
					Description: r.Doc.Description,
				}
			}

			if r.RequestType != nil {
				typeName := r.RequestType.TypeName()
				if ts, ok := typeMap[typeName]; ok {
					route.RequestType = ts
				} else {
					route.RequestType = &TypeSpec{Name: typeName}
				}
			}

			if r.ResponseType != nil {
				typeName := r.ResponseType.TypeName()
				// Handle array response
				if strings.HasPrefix(typeName, "[]") {
					elemType := strings.TrimPrefix(typeName, "[]")
					if ts, ok := typeMap[elemType]; ok {
						route.ResponseType = &TypeSpec{Name: "[]" + ts.Name, Members: ts.Members}
					} else {
						route.ResponseType = &TypeSpec{Name: typeName}
					}
				} else if ts, ok := typeMap[typeName]; ok {
					route.ResponseType = ts
				} else {
					route.ResponseType = &TypeSpec{Name: typeName}
				}
			}

			group.Routes = append(group.Routes, route)
		}

		spec.Groups = append(spec.Groups, group)
	}

	return spec, nil
}

func convertMember(m ast.Member) MemberSpec {
	ms := MemberSpec{
		Name:     m.Name,
		IsInline: m.IsInline,
		Comment:  m.Comment,
		Docs:     m.Docs,
	}

	if m.Type != nil {
		ms.Type = getBaseTypeName(m.Type)
		ms.RawType = m.Type.TypeName()
	}

	// Parse struct tag
	tags := ParseTag(m.Tag)
	ms.JsonTag = tags["json"]
	ms.PathTag = tags["path"]
	ms.QueryTag = tags["query"]
	ms.FormTag = tags["form"]
	ms.HeaderTag = tags["header"]
	ms.Validate = tags["validate"]
	ms.Default = tags["default"]

	// Parse validation rules (range and options)
	ms.Range, ms.RangeMin, ms.RangeMax, ms.Options = parseValidationRules(ms.Validate)

	// Check for optional in json tag
	if ms.JsonTag != "" && strings.Contains(ms.JsonTag, ",optional") {
		ms.Optional = true
		ms.JsonTag = strings.Replace(ms.JsonTag, ",optional", "", 1)
	}

	// Check for default in json/query/form tags
	for _, tag := range []string{ms.JsonTag, ms.QueryTag, ms.FormTag} {
		if idx := strings.Index(tag, ",default="); idx != -1 {
			// Extract default value
			rest := tag[idx+9:]
			endIdx := strings.Index(rest, ",")
			if endIdx == -1 {
				ms.Default = rest
			} else {
				ms.Default = rest[:endIdx]
			}
		}
	}

	// Check for optional in other tags
	for _, tag := range []string{ms.PathTag, ms.QueryTag, ms.FormTag, ms.HeaderTag} {
		if strings.Contains(tag, ",optional") {
			ms.Optional = true
		}
	}

	return ms
}

func getBaseTypeName(t ast.TypeExpr) string {
	switch v := t.(type) {
	case *ast.IdentType:
		return v.Name
	case *ast.ArrayType:
		return getBaseTypeName(v.Element)
	case *ast.PointerType:
		return getBaseTypeName(v.Element)
	case *ast.MapType:
		return "map"
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.StructType:
		return "struct"
	default:
		return ""
	}
}

// GetAllRoutes returns all routes from all groups
func (s *ServiceSpec) GetAllRoutes() []RouteSpec {
	var routes []RouteSpec
	for _, g := range s.Groups {
		routes = append(routes, g.Routes...)
	}
	return routes
}

// GetTypeByName returns a type by name
func (s *ServiceSpec) GetTypeByName(name string) *TypeSpec {
	for i := range s.Types {
		if s.Types[i].Name == name {
			return &s.Types[i]
		}
	}
	return nil
}

// GetGroupByName returns a group by name (from annotation.group)
func (s *ServiceSpec) GetGroupByName(name string) *Group {
	for i := range s.Groups {
		if s.Groups[i].Annotation != nil && s.Groups[i].Annotation.Group == name {
			return &s.Groups[i]
		}
	}
	return nil
}

// IsBasicType returns true if the type is a basic Go type
func IsBasicType(typeName string) bool {
	switch typeName {
	case "bool",
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64",
		"complex64", "complex128",
		"string", "byte", "rune",
		"interface{}", "any":
		return true
	}
	return false
}

// GoType returns the Go type string for a member
func (m *MemberSpec) GoType() string {
	if m.RawType != "" {
		return m.RawType
	}
	return m.Type
}

// FullPath returns the full path including prefix
func (r *RouteSpec) FullPath(prefix string) string {
	if prefix == "" {
		return r.Path
	}
	prefix = strings.TrimSuffix(prefix, "/")
	path := r.Path
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return prefix + path
}

// HasRequestBody returns true if the route expects a request body
func (r *RouteSpec) HasRequestBody() bool {
	if r.RequestType == nil {
		return false
	}
	// GET, HEAD, OPTIONS typically don't have request bodies
	switch r.Method {
	case "GET", "HEAD", "OPTIONS", "DELETE":
		return false
	}
	return true
}

// HasResponseBody returns true if the route returns a response body
func (r *RouteSpec) HasResponseBody() bool {
	return r.ResponseType != nil
}
