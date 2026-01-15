// Package openapi provides OpenAPI 3.0 spec generation from .api files
package openapi

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	ast "github.com/ssgohq/ssgo/internal/ast/api"
	spec "github.com/ssgohq/ssgo/tool/internal/spec/api"
)

// Generator generates OpenAPI specs from parsed API specs
type Generator struct {
	apiSpec     *ast.ApiSpec
	serviceSpec *spec.ServiceSpec
	output      string
	format      string // json or yaml
}

// NewGenerator creates a new OpenAPI generator
func NewGenerator(apiSpec *ast.ApiSpec, serviceSpec *spec.ServiceSpec, output, format string) *Generator {
	return &Generator{
		apiSpec:     apiSpec,
		serviceSpec: serviceSpec,
		output:      output,
		format:      format,
	}
}

// Generate generates the OpenAPI specification and writes it to a file
func (g *Generator) Generate() error {
	openapi := g.buildOpenAPISpec()

	var data []byte
	var err error

	if g.format == "yaml" || g.format == "yml" {
		data, err = yaml.Marshal(openapi)
	} else {
		data, err = json.MarshalIndent(openapi, "", "  ")
	}

	if err != nil {
		return fmt.Errorf("failed to marshal OpenAPI spec: %w", err)
	}

	if err := os.MkdirAll(g.output, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	filename := "openapi." + g.format
	outputPath := filepath.Join(g.output, filename)

	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write OpenAPI spec: %w", err)
	}

	return nil
}

// buildOpenAPISpec builds the complete OpenAPI specification
func (g *Generator) buildOpenAPISpec() *OpenAPISpec {
	spec := &OpenAPISpec{
		OpenAPI: "3.0.3",
		Info:    g.buildInfo(),
		Paths:   g.buildPaths(),
		Components: &Components{
			Schemas:         g.buildSchemas(),
			SecuritySchemes: g.buildSecuritySchemes(),
		},
		Tags: g.buildTags(),
	}

	return spec
}

// buildInfo builds the OpenAPI info object from API info
func (g *Generator) buildInfo() Info {
	info := Info{
		Title:   "API",
		Version: "1.0.0",
	}

	if g.apiSpec.Info != nil {
		if g.apiSpec.Info.Title != "" {
			info.Title = g.apiSpec.Info.Title
		}
		if g.apiSpec.Info.Desc != "" {
			info.Description = g.apiSpec.Info.Desc
		}
		if g.apiSpec.Info.Version != "" {
			info.Version = g.apiSpec.Info.Version
		}
		if g.apiSpec.Info.Author != "" || g.apiSpec.Info.Email != "" {
			info.Contact = &Contact{
				Name:  g.apiSpec.Info.Author,
				Email: g.apiSpec.Info.Email,
			}
		}
	}

	return info
}

// buildTags builds OpenAPI tags from service groups
func (g *Generator) buildTags() []Tag {
	tagMap := make(map[string]bool)
	var tags []Tag

	for _, group := range g.serviceSpec.Groups {
		if group.Annotation != nil && group.Annotation.Group != "" {
			if !tagMap[group.Annotation.Group] {
				tagMap[group.Annotation.Group] = true
				tags = append(tags, Tag{
					Name: group.Annotation.Group,
				})
			}
		}
	}

	return tags
}

// buildPaths builds OpenAPI paths from routes
func (g *Generator) buildPaths() map[string]PathItem {
	paths := make(map[string]PathItem)

	for _, group := range g.serviceSpec.Groups {
		for _, route := range group.Routes {
			fullPath := route.FullPath(getPrefix(group))
			openAPIPath := convertPathParams(fullPath)

			pathItem, exists := paths[openAPIPath]
			if !exists {
				pathItem = PathItem{}
			}

			operation := g.buildOperation(&route, &group)

			switch strings.ToUpper(route.Method) {
			case "GET":
				pathItem.Get = operation
			case "POST":
				pathItem.Post = operation
			case "PUT":
				pathItem.Put = operation
			case "DELETE":
				pathItem.Delete = operation
			case "PATCH":
				pathItem.Patch = operation
			case "HEAD":
				pathItem.Head = operation
			case "OPTIONS":
				pathItem.Options = operation
			}

			paths[openAPIPath] = pathItem
		}
	}

	return paths
}

// buildOperation builds an OpenAPI operation from a route
func (g *Generator) buildOperation(route *spec.RouteSpec, group *spec.Group) *Operation {
	operation := &Operation{
		OperationID: route.Handler,
		Responses:   make(map[string]Response),
	}

	if group.Annotation != nil && group.Annotation.Group != "" {
		operation.Tags = []string{group.Annotation.Group}
	}

	if route.Doc != nil {
		operation.Summary = route.Doc.Summary
		operation.Description = route.Doc.Description
	}

	operation.Parameters = g.buildParameters(route)

	if route.HasRequestBody() && route.RequestType != nil {
		operation.RequestBody = g.buildRequestBody(route.RequestType)
	}

	operation.Responses["200"] = g.buildSuccessResponse(route)

	operation.Responses["400"] = Response{
		Description: "Bad Request",
	}
	operation.Responses["500"] = Response{
		Description: "Internal Server Error",
	}

	if group.Annotation != nil && group.Annotation.JWT != "" {
		operation.Security = []SecurityRequirement{
			{"BearerAuth": []string{}},
		}
		operation.Responses["401"] = Response{
			Description: "Unauthorized",
		}
	}

	return operation
}

// buildParameters builds OpenAPI parameters from route request type
func (g *Generator) buildParameters(route *spec.RouteSpec) []Parameter {
	var params []Parameter

	if route.RequestType == nil {
		return params
	}

	for _, member := range route.RequestType.Members {
		if member.PathTag != "" {
			paramName := extractTagName(member.PathTag)
			schema := g.memberToSchema(&member)
			params = append(params, Parameter{
				Name:     paramName,
				In:       "path",
				Required: true,
				Schema:   schema,
			})
		}

		if member.QueryTag != "" {
			paramName := extractTagName(member.QueryTag)
			schema := g.memberToSchema(&member)
			params = append(params, Parameter{
				Name:     paramName,
				In:       "query",
				Required: !member.Optional && !strings.Contains(member.QueryTag, "optional"),
				Schema:   schema,
			})
		}

		if member.HeaderTag != "" {
			paramName := extractTagName(member.HeaderTag)
			schema := g.memberToSchema(&member)
			params = append(params, Parameter{
				Name:     paramName,
				In:       "header",
				Required: !member.Optional && !strings.Contains(member.HeaderTag, "optional"),
				Schema:   schema,
			})
		}
	}

	return params
}

// buildRequestBody builds OpenAPI request body from type
func (g *Generator) buildRequestBody(typeSpec *spec.TypeSpec) *RequestBody {
	if typeSpec == nil {
		return nil
	}

	typeName := typeSpec.Name
	if strings.HasPrefix(typeName, "[]") {
		elemType := strings.TrimPrefix(typeName, "[]")
		return &RequestBody{
			Required: true,
			Content: map[string]MediaType{
				"application/json": {
					Schema: ArraySchema(SchemaRef(elemType)),
				},
			},
		}
	}

	return &RequestBody{
		Required: true,
		Content: map[string]MediaType{
			"application/json": {
				Schema: SchemaRef(typeName),
			},
		},
	}
}

// buildSuccessResponse builds the success response for a route
func (g *Generator) buildSuccessResponse(route *spec.RouteSpec) Response {
	resp := Response{
		Description: "Successful response",
	}

	if route.ResponseType != nil {
		typeName := route.ResponseType.Name
		var schema *Schema

		if strings.HasPrefix(typeName, "[]") {
			elemType := strings.TrimPrefix(typeName, "[]")
			schema = ArraySchema(SchemaRef(elemType))
		} else {
			schema = SchemaRef(typeName)
		}

		resp.Content = map[string]MediaType{
			"application/json": {
				Schema: schema,
			},
		}
	}

	return resp
}

// buildSchemas builds OpenAPI component schemas from types
func (g *Generator) buildSchemas() map[string]*Schema {
	schemas := make(map[string]*Schema)

	for _, typeSpec := range g.serviceSpec.Types {
		schema := g.typeToSchema(&typeSpec)
		schemas[typeSpec.Name] = schema
	}

	return schemas
}

// typeToSchema converts a type spec to an OpenAPI schema
func (g *Generator) typeToSchema(t *spec.TypeSpec) *Schema {
	schema := ObjectSchema()
	var required []string

	for _, member := range t.Members {
		if member.IsInline {
			continue
		}

		memberSchema := g.memberToSchema(&member)

		propName := member.Name
		if member.JsonTag != "" {
			propName = extractTagName(member.JsonTag)
		}

		if propName == "" || propName == "-" {
			continue
		}

		schema.Properties[propName] = memberSchema

		isRequired := !member.Optional
		if member.Validate != "" {
			constraints := ParseValidateTag(member.Validate)
			if constraints != nil && constraints.Required {
				isRequired = true
			}
		}
		if isRequired {
			required = append(required, propName)
		}
	}

	if len(required) > 0 {
		schema.Required = required
	}

	return schema
}

// memberToSchema converts a member spec to an OpenAPI schema
func (g *Generator) memberToSchema(member *spec.MemberSpec) *Schema {
	schema := &Schema{}
	goType := member.GoType()

	if strings.HasPrefix(goType, "[]") {
		elemType := strings.TrimPrefix(goType, "[]")
		if IsBasicType(elemType) {
			schemaType, format := GoTypeToOpenAPI(elemType)
			schema.Type = "array"
			schema.Items = &Schema{Type: schemaType, Format: format}
		} else {
			schema.Type = "array"
			schema.Items = SchemaRef(elemType)
		}
		return schema
	}

	if strings.HasPrefix(goType, "map[") {
		schema.Type = "object"
		schema.AdditionalProperties = &Schema{Type: "object"}
		return schema
	}

	if strings.HasPrefix(goType, "*") {
		goType = strings.TrimPrefix(goType, "*")
		schema.Nullable = true
	}

	if IsBasicType(goType) {
		schemaType, format := GoTypeToOpenAPI(goType)
		schema.Type = schemaType
		if format != "" {
			schema.Format = format
		}

		if member.Validate != "" {
			constraints := ParseValidateTag(member.Validate)
			ApplyConstraintsToSchema(schema, constraints, schemaType)
		}

		if member.Default != "" {
			schema.Default = member.Default
		}

		if member.JsonTag != "" {
			if options := ParseOptionsTag(member.JsonTag); len(options) > 0 {
				for _, opt := range options {
					schema.Enum = append(schema.Enum, opt)
				}
			}
			if def := ParseDefaultValue(member.JsonTag); def != "" {
				schema.Default = def
			}
		}

		return schema
	}

	return SchemaRef(goType)
}

// buildSecuritySchemes builds OpenAPI security schemes
func (g *Generator) buildSecuritySchemes() map[string]*SecurityScheme {
	schemes := make(map[string]*SecurityScheme)

	hasJWT := false
	for _, group := range g.serviceSpec.Groups {
		if group.Annotation != nil && group.Annotation.JWT != "" {
			hasJWT = true
			break
		}
	}

	if hasJWT {
		schemes["BearerAuth"] = &SecurityScheme{
			Type:         "http",
			Scheme:       "bearer",
			BearerFormat: "JWT",
			Description:  "JWT Bearer token authentication",
		}
	}

	return schemes
}

// getPrefix returns the prefix from group annotation
func getPrefix(group spec.Group) string {
	if group.Annotation != nil {
		return group.Annotation.Prefix
	}
	return ""
}

// convertPathParams converts :param style to {param} style
func convertPathParams(path string) string {
	re := regexp.MustCompile(`:(\w+)`)
	return re.ReplaceAllString(path, "{$1}")
}

// extractTagName extracts the field name from a tag value
func extractTagName(tag string) string {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return ""
	}

	parts := strings.Split(tag, ",")
	name := strings.TrimSpace(parts[0])
	name = strings.Trim(name, "\"'")

	return name
}

// GenerateFromFile is a convenience function to generate OpenAPI from a file
func GenerateFromFile(apiFile, outputDir, format string) error {
	apiSpec, err := ast.Parse(apiFile)
	if err != nil {
		return fmt.Errorf("failed to parse API file: %w", err)
	}

	apiSpec, err = ast.ResolveImports(apiSpec)
	if err != nil {
		return fmt.Errorf("failed to resolve imports: %w", err)
	}

	serviceSpec, err := spec.FromAST(apiSpec)
	if err != nil {
		return fmt.Errorf("failed to convert to service spec: %w", err)
	}

	gen := NewGenerator(apiSpec, serviceSpec, outputDir, format)
	return gen.Generate()
}
