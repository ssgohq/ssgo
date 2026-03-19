package hertz

import (
	"strings"

	"github.com/ssgohq/ssgo/internal/util/naming"
	spec "github.com/ssgohq/ssgo/tool/internal/spec/api"
)

// RPCClientTemplateData represents RPC client data for templates
type RPCClientTemplateData struct {
	Name         string // Field name (e.g., "User")
	ServiceName  string // Service name for comments
	Package      string // Proto package name (e.g., "user")
	ServiceLower string // Lowercase service name for kitex_gen (e.g., "userservice")
	TypesModule  string // Module path for kitex_gen imports
}

// APITypeData represents a single type
type APITypeData struct {
	Name    string
	Comment string
	Fields  []APIFieldData
	IsAlias bool   // true if this is a type alias
	AliasOf string // the aliased type name (only set if IsAlias is true)
}

// APIFieldData represents a field in a type
type APIFieldData struct {
	Name         string
	Type         string
	JSONTag      string
	Validate     string
	Comment      string
	Required     bool     // field is required
	Optional     bool     // field is optional
	Default      string   // default value string
	DefaultValue string   // default value with proper formatting for code gen
	ZeroValue    string   // zero value for the type
	Range        string   // range constraint e.g., "1:100"
	RangeMin     *int64   // minimum value
	RangeMax     *int64   // maximum value
	Options      []string // allowed values
}

// APIData is the root data structure passed to Hertz API templates.
type APIData struct {
	Module        string
	ServiceName   string
	Port          int
	GoVersion     string
	WithTrace     bool
	WithDB        string
	WithRedis     bool
	TypesModule   string
	HasRPCClients bool
	RPCClients    []RPCClientTemplateData

	// Spec data
	Types      []APITypeData
	Groups     []APIRouteGroupData
	Middleware []string
	Imports    []string

	// Current loop iteration context (set during loop expansion)
	CurrentGroup *APIRouteGroupData
	CurrentRoute *APIRouteData
}

// APIRouteGroupData represents a route group for templates.
type APIRouteGroupData struct {
	Name       string
	VarName    string
	Prefix     string
	Middleware []string
	Routes     []APIRouteData
}

// GetName returns the group name.
func (g *APIRouteGroupData) GetName() string {
	return g.Name
}

// APIRouteData represents a single route for templates.
type APIRouteData struct {
	Method             string
	Path               string
	FullPath           string
	Handler            string
	HandlerName        string
	Package            string
	Group              string
	LogicPackage       string
	LogicName          string
	LogicMethod        string
	Tag                string
	Summary            string
	Doc                string
	RequestType        string
	ResponseType       string
	HasRequest         bool
	HasResponse        bool
	HasPathParams      bool
	PathParamName      string
	PathParamFieldName string
	PathParamType      string
	PathParamGoType    string
	PathParamDoc       string
}

// GetName returns the handler name.
func (r *APIRouteData) GetName() string {
	return r.HandlerName
}

// GetRequestType returns the request type.
func (r *APIRouteData) GetRequestType() string {
	return r.RequestType
}

// GetResponseType returns the response type.
func (r *APIRouteData) GetResponseType() string {
	return r.ResponseType
}

// Template accessor methods for current loop context

// Group returns the current group name for templates.
func (d *APIData) Group() string {
	if d.CurrentGroup != nil {
		return d.CurrentGroup.Name
	}
	return ""
}

// HandlerName returns the current route's handler name for templates.
func (d *APIData) HandlerName() string {
	if d.CurrentRoute != nil {
		return d.CurrentRoute.HandlerName
	}
	return ""
}

// LogicName returns the current route's logic name for templates.
func (d *APIData) LogicName() string {
	if d.CurrentRoute != nil {
		return d.CurrentRoute.LogicName
	}
	return ""
}

// LogicMethod returns the current route's logic method for templates.
func (d *APIData) LogicMethod() string {
	if d.CurrentRoute != nil {
		return d.CurrentRoute.LogicMethod
	}
	return ""
}

// Package returns the current route's package for templates.
func (d *APIData) Package() string {
	if d.CurrentRoute != nil {
		return d.CurrentRoute.Package
	}
	return ""
}

// HasPathParams returns whether the current route has path parameters.
func (d *APIData) HasPathParams() bool {
	if d.CurrentRoute != nil {
		return d.CurrentRoute.HasPathParams
	}
	return false
}

// PathParamName returns the current route's path param name.
func (d *APIData) PathParamName() string {
	if d.CurrentRoute != nil {
		return d.CurrentRoute.PathParamName
	}
	return ""
}

// RequestType returns the current route's request type.
func (d *APIData) RequestType() string {
	if d.CurrentRoute != nil {
		return d.CurrentRoute.RequestType
	}
	return ""
}

// ResponseType returns the current route's response type.
func (d *APIData) ResponseType() string {
	if d.CurrentRoute != nil {
		return d.CurrentRoute.ResponseType
	}
	return ""
}

// HasRequest returns whether the current route has a request type.
func (d *APIData) HasRequest() bool {
	if d.CurrentRoute != nil {
		return d.CurrentRoute.HasRequest
	}
	return false
}

// HasResponse returns whether the current route has a response type.
func (d *APIData) HasResponse() bool {
	if d.CurrentRoute != nil {
		return d.CurrentRoute.HasResponse
	}
	return false
}

// LogicPackage returns the current route's logic package.
func (d *APIData) LogicPackage() string {
	if d.CurrentRoute != nil {
		return d.CurrentRoute.LogicPackage
	}
	return ""
}

// BuildAPIData builds APIData from spec and options.
func BuildAPIData(apiSpec *spec.ServiceSpec, opts APIOptions) *APIData {
	data := &APIData{
		Module:        opts.Module,
		ServiceName:   opts.ServiceName,
		Port:          opts.Port,
		GoVersion:     "1.23",
		WithTrace:     opts.WithTrace,
		WithDB:        opts.WithDB,
		WithRedis:     opts.WithRedis,
		HasRPCClients: len(opts.RPCClients) > 0,
	}

	if data.ServiceName == "" && apiSpec != nil {
		data.ServiceName = apiSpec.Name
	}
	if data.Port == 0 {
		data.Port = 8080
	}

	// Build RPC clients
	data.RPCClients = make([]RPCClientTemplateData, len(opts.RPCClients))
	for i, c := range opts.RPCClients {
		data.RPCClients[i] = RPCClientTemplateData{
			Name:         c.Name,
			ServiceName:  c.ServiceName,
			Package:      c.Package,
			ServiceLower: c.ServiceLower,
			TypesModule:  c.GetTypesModule(),
		}
	}

	if len(opts.RPCClients) > 0 {
		data.TypesModule = opts.RPCClients[0].GetTypesModule()
	}

	if apiSpec == nil {
		return data
	}

	// Build types
	data.Types = buildTypesFromSpec(apiSpec)

	// Build groups and routes
	var groups []APIRouteGroupData
	imports := make(map[string]bool)
	middleware := make(map[string]bool)

	for _, group := range apiSpec.Groups {
		groupName := getGroupName(&group)
		groupPrefix := getGroupPrefix(&group)
		groupMiddleware := getGroupMiddleware(&group)

		// Collect middleware
		for _, mw := range groupMiddleware {
			middleware[mw] = true
		}

		rg := APIRouteGroupData{
			Name:       groupName,
			VarName:    naming.ToSnakeCase(groupName) + "Group",
			Prefix:     groupPrefix,
			Middleware: groupMiddleware,
		}

		if groupName == "" {
			rg.VarName = "rootGroup"
		}

		for _, route := range group.Routes {
			packageName := groupName
			if packageName == "" {
				packageName = "handler"
			}

			r := APIRouteData{
				Method:       route.Method,
				Path:         route.Path,
				FullPath:     groupPrefix + route.Path,
				Handler:      route.Handler,
				HandlerName:  naming.HandlerName(route.Handler),
				Package:      naming.SanitizePackageName(packageName),
				Group:        groupName,
				LogicPackage: logicPackage(groupName),
				LogicName:    naming.LogicName(route.Handler),
				LogicMethod:  logicMethod(route.Handler),
				HasRequest:   route.RequestType != nil,
				HasResponse:  route.ResponseType != nil,
			}

			if route.RequestType != nil {
				r.RequestType = naming.CleanTypeName(route.RequestType.Name)
			}
			if route.ResponseType != nil {
				r.ResponseType = naming.CleanTypeName(route.ResponseType.Name)
			}

			// Extract path param info
			if paramName := extractPathParam(route.Path); paramName != "" {
				r.HasPathParams = true
				r.PathParamName = paramName
				r.PathParamFieldName = naming.ToPascalCase(paramName)
				r.PathParamType = "integer"
				r.PathParamGoType = "int64"
			}

			// Extract doc info from route
			if route.Doc != nil {
				r.Tag = groupName // Tag comes from group, not route
				r.Summary = route.Doc.Summary
				r.Doc = route.Doc.Description
			}

			rg.Routes = append(rg.Routes, r)

			// Add import
			if groupName != "" {
				importPath := opts.Module + "/internal/api/handler/" + naming.ToSnakeCase(groupName)
				imports[importPath] = true
			}
		}

		groups = append(groups, rg)
	}

	data.Groups = groups

	// Convert maps to slices
	for imp := range imports {
		data.Imports = append(data.Imports, imp)
	}
	for mw := range middleware {
		data.Middleware = append(data.Middleware, mw)
	}

	return data
}

// buildTypesFromSpec extracts types from API spec.
func buildTypesFromSpec(apiSpec *spec.ServiceSpec) []APITypeData {
	var types []APITypeData

	for _, t := range apiSpec.Types {
		apiType := APITypeData{
			Name:    t.Name,
			IsAlias: t.IsAlias,
			AliasOf: t.AliasOf,
		}

		// Only add fields for non-alias types
		if !t.IsAlias {
			for _, m := range t.Members {
				field := APIFieldData{
					Name:      m.Name,
					Type:      m.GoType(),
					JSONTag:   getJSONTagValue(m),
					Validate:  m.Validate,
					Comment:   m.Comment,
					Required:  !m.Optional && m.Validate != "" && containsRequired(m.Validate),
					Optional:  m.Optional,
					Default:   m.Default,
					Range:     m.Range,
					RangeMin:  m.RangeMin,
					RangeMax:  m.RangeMax,
					Options:   m.Options,
					ZeroValue: getZeroValue(m.GoType()),
				}

				if field.Default != "" {
					field.DefaultValue = getDefaultValue(field.Default, field.Type)
				}

				apiType.Fields = append(apiType.Fields, field)
			}
		}

		types = append(types, apiType)
	}

	return types
}

// containsRequired checks if validate tag contains "required".
func containsRequired(validate string) bool {
	return validate == "required" ||
		strings.HasPrefix(validate, "required,") ||
		strings.HasSuffix(validate, ",required") ||
		strings.Contains(validate, ",required,")
}

// extractPathParam extracts the first path parameter from a route path.
func extractPathParam(path string) string {
	for _, part := range splitPath(path) {
		if len(part) > 1 && part[0] == ':' {
			return part[1:]
		}
		if len(part) > 2 && part[0] == '{' && part[len(part)-1] == '}' {
			return part[1 : len(part)-1]
		}
	}
	return ""
}

// splitPath splits a path by / and filters empty parts.
func splitPath(path string) []string {
	parts := strings.Split(path, "/")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

// logicPackage generates logic package name from group.
func logicPackage(group string) string {
	if group == "" {
		return "logic"
	}
	return naming.SanitizePackageName(group)
}

// logicMethod generates logic method name from handler.
func logicMethod(handler string) string {
	return naming.ToPascalCase(handler)
}
