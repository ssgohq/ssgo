// Package hertz provides code generation for Hertz HTTP server from .api spec
package hertz

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ssgohq/ssgo/tool/internal/gen"
	"github.com/ssgohq/ssgo/tool/internal/generator/templates"
	spec "github.com/ssgohq/ssgo/tool/internal/spec/api"
	"github.com/ssgohq/ssgo/internal/util/naming"
)

// Options represents basic generator options
type Options struct {
	Output   string // Output directory
	Module   string // Go module name
	Style    string // Naming style
	UseTypes string // Import path for shared types module (optional)
}

// RPCClientConfig represents an RPC client configuration
type RPCClientConfig struct {
	Name         string // Client name (e.g., "User")
	ServiceName  string // Full service name (e.g., "UserService")
	DMSModule    string // DMS module path for RPC server (e.g., "github.com/org/user-rpc")
	ModelModule  string // Model module path for kitex_gen types (e.g., "github.com/org/common-pb")
	Package      string // Proto package name (e.g., "user")
	ServiceLower string // Lowercase service name for kitex_gen (e.g., "userservice")
}

// GetTypesModule returns the module path for importing types (kitex_gen)
// If ModelModule is set, use it; otherwise fall back to DMSModule
func (c *RPCClientConfig) GetTypesModule() string {
	if c.ModelModule != "" {
		return c.ModelModule
	}
	return c.DMSModule
}

// APIOptions extends Options with API-specific settings
type APIOptions struct {
	Options

	// WithLogic enables logic layer generation
	WithLogic bool

	// RPCClients specifies RPC clients to generate
	RPCClients []RPCClientConfig

	// ServiceName is the name of the API service
	ServiceName string

	// Port is the default API port
	Port int

	// Optional components for generated code
	WithTrace bool   // Enable OpenTelemetry tracing config
	WithDB    string // Add database config ("postgres" or "mysql")
	WithRedis bool   // Add Redis config
}

// APIGenerator generates Hertz API code with ServiceContext and RPC integration.
// It implements the gen.Generator interface for use with SDK's Runner pattern.
type APIGenerator struct {
	spec *spec.ServiceSpec
	opts APIOptions
	data *APIData
}

// NewAPIGenerator creates a new APIGenerator.
func NewAPIGenerator(apiSpec *spec.ServiceSpec, opts APIOptions) *APIGenerator {
	return &APIGenerator{
		spec: apiSpec,
		opts: opts,
		data: BuildAPIData(apiSpec, opts),
	}
}

// Name returns the generator name.
func (g *APIGenerator) Name() string {
	return "hertz-api"
}

// Steps returns the list of generation steps.
func (g *APIGenerator) Steps() []gen.Step {
	return []gen.Step{
		{Name: "directories", Run: g.stepDirectories, Tags: []string{"scaffold"}},
		{Name: "types", Run: g.stepTypes, Tags: []string{"scaffold", "types"}},
		{Name: "config", Run: g.stepConfig, Tags: []string{"scaffold", "config"}},
		{Name: "service context", Run: g.stepServiceContext, Tags: []string{"scaffold"}},
		{Name: "middleware", Run: g.stepMiddleware, Tags: []string{"scaffold"}},
		{Name: "httputil", Run: g.stepHTTPUtil, Tags: []string{"scaffold"}},
		{Name: "handlers", Run: g.stepHandlers, Tags: []string{"scaffold", "handlers"}},
		{Name: "logic", Run: g.stepLogic, Tags: []string{"scaffold", "logic"}},
		{Name: "routes", Run: g.stepRoutes, Tags: []string{"scaffold", "routes"}},
		{Name: "main", Run: g.stepMain, Tags: []string{"scaffold"}},
		{Name: "config yaml", Run: g.stepConfigYAML, Tags: []string{"scaffold", "config"}},
		{Name: "go.mod", Run: g.stepGoMod, Tags: []string{"scaffold"}},
	}
}

// Generate generates all API code files using the Runner pattern.
func (g *APIGenerator) Generate() error {
	ctx := context.Background()

	fmt.Printf("Generating API server code with ServiceContext pattern...\n")
	fmt.Printf("  Output:  %s\n", g.opts.Output)
	fmt.Printf("  Module:  %s\n", g.opts.Module)
	fmt.Printf("  Service: %s\n", g.opts.ServiceName)
	if len(g.opts.RPCClients) > 0 {
		fmt.Printf("  RPC Clients:\n")
		for _, c := range g.opts.RPCClients {
			fmt.Printf("    - %s (%s)\n", c.Name, c.DMSModule)
		}
	}
	fmt.Println()

	runner := g.newRunner()
	if err := runner.Run(ctx, g); err != nil {
		return err
	}

	g.printGeneratedStructure()
	return nil
}

// GenerateLogicOnly generates only the logic layer files.
func (g *APIGenerator) GenerateLogicOnly() error {
	ctx := context.Background()

	fmt.Printf("Generating logic files only...\n")
	fmt.Printf("  Output: %s\n", g.opts.Output)
	fmt.Printf("  Module: %s\n", g.opts.Module)
	fmt.Println()

	runner := g.newRunner()
	if err := runner.RunWithTags(ctx, g, "logic"); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("Logic generation completed successfully!")
	return nil
}

// newRunner creates a new Runner configured for this generator.
func (g *APIGenerator) newRunner() *gen.Runner {
	funcMap := map[string]any{
		"ToSnakeCase":     naming.ToSnakeCase,
		"ToCamelCase":     naming.ToCamelCase,
		"ToPascalCase":    naming.ToPascalCase,
		"ToKebabCase":     naming.ToKebabCase,
		"HandlerName":     naming.HandlerName,
		"BaseHandlerName": naming.BaseHandlerName,
		"LogicName":       naming.LogicName,
		"CleanTypeName":   naming.CleanTypeName,
		"lower":           strings.ToLower,
		"upper":           strings.ToUpper,
		"title":           strings.Title,
	}

	return gen.NewRunner(gen.RunnerConfig{
		Options: gen.CommonOptions{
			OutputDir: g.opts.Output,
			Module:    g.opts.Module,
			Verbose:   false,
			WithTrace: g.opts.WithTrace,
			WithDB:    g.opts.WithDB,
			WithRedis: g.opts.WithRedis,
		},
		TemplatesFS: templates.HertzTemplates,
		TemplateDir: "hertz",
		FuncMap:     funcMap,
	})
}

// Step implementations

func (g *APIGenerator) stepDirectories(_ context.Context, r *gen.Runner) error {
	dirs := g.buildDirectories()
	absDirs := make([]string, len(dirs))
	for i, d := range dirs {
		absDirs[i] = filepath.Join(r.Opt.OutputDir, d)
	}
	return r.Files.CreateDirs(absDirs)
}

func (g *APIGenerator) stepTypes(_ context.Context, r *gen.Runner) error {
	return r.Tpl.RenderToFile(r.Files, "api_types.tpl",
		filepath.Join(r.Opt.OutputDir, "internal/types/types.go"),
		map[string]any{
			"Module": g.data.Module,
			"Types":  g.data.Types,
		})
}

func (g *APIGenerator) stepConfig(_ context.Context, r *gen.Runner) error {
	return r.Tpl.RenderToFile(r.Files, "api_config.tpl",
		filepath.Join(r.Opt.OutputDir, "internal/config/config.go"),
		map[string]any{
			"Module":        g.data.Module,
			"HasRPCClients": g.data.HasRPCClients,
			"RPCClients":    g.data.RPCClients,
			"WithTrace":     g.data.WithTrace,
			"WithDB":        g.data.WithDB,
			"WithRedis":     g.data.WithRedis,
		})
}

func (g *APIGenerator) stepServiceContext(_ context.Context, r *gen.Runner) error {
	return r.Tpl.RenderToFile(r.Files, "api_svc.tpl",
		filepath.Join(r.Opt.OutputDir, "internal/svc/service_context.go"),
		map[string]any{
			"Module":        g.data.Module,
			"TypesModule":   g.data.TypesModule,
			"HasRPCClients": g.data.HasRPCClients,
			"RPCClients":    g.data.RPCClients,
		})
}

func (g *APIGenerator) stepMiddleware(_ context.Context, r *gen.Runner) error {
	return r.Tpl.RenderToFile(r.Files, "api_middleware.tpl",
		filepath.Join(r.Opt.OutputDir, "internal/middleware/middleware.go"),
		map[string]any{
			"Module":     g.data.Module,
			"Middleware": g.data.Middleware,
		})
}

func (g *APIGenerator) stepHTTPUtil(_ context.Context, r *gen.Runner) error {
	return r.Tpl.RenderToFile(r.Files, "api_httputil_errors.tpl",
		filepath.Join(r.Opt.OutputDir, "internal/pkg/httputil/errors.go"),
		map[string]any{
			"Module": g.data.Module,
		})
}

func (g *APIGenerator) stepHandlers(_ context.Context, r *gen.Runner) error {
	for i := range g.data.Groups {
		group := &g.data.Groups[i]
		for j := range group.Routes {
			route := &group.Routes[j]

			handlerData := map[string]any{
				"Package":            route.Package,
				"Module":             g.data.Module,
				"Group":              route.Group,
				"HandlerName":        route.HandlerName,
				"LogicPackage":       route.LogicPackage,
				"LogicName":          route.LogicName,
				"LogicMethod":        route.LogicMethod,
				"Method":             route.Method,
				"Path":               route.Path,
				"FullPath":           route.FullPath,
				"Tag":                route.Tag,
				"Summary":            route.Summary,
				"Doc":                route.Doc,
				"RequestType":        route.RequestType,
				"ResponseType":       route.ResponseType,
				"HasRequest":         route.HasRequest,
				"HasResponse":        route.HasResponse,
				"HasPathParams":      route.HasPathParams,
				"PathParamName":      route.PathParamName,
				"PathParamFieldName": route.PathParamFieldName,
				"PathParamType":      route.PathParamType,
				"PathParamGoType":    route.PathParamGoType,
				"PathParamDoc":       route.PathParamDoc,
			}

			fileName := naming.FileNameFromHandler(route.Handler) + "_handler.go"
			var path string
			if group.Name != "" {
				path = filepath.Join(r.Opt.OutputDir, "internal", "handler", naming.ToSnakeCase(group.Name), fileName)
			} else {
				path = filepath.Join(r.Opt.OutputDir, "internal", "handler", fileName)
			}

			if err := r.Tpl.RenderToFile(r.Files, "api_handler.tpl", path, handlerData); err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *APIGenerator) stepLogic(_ context.Context, r *gen.Runner) error {
	for i := range g.data.Groups {
		group := &g.data.Groups[i]
		for j := range group.Routes {
			route := &group.Routes[j]

			packageName := group.Name
			if packageName == "" {
				packageName = "logic"
			}

			var rpcClient, rpcPackage string
			if g.data.HasRPCClients && len(g.opts.RPCClients) > 0 {
				rpcClient = g.opts.RPCClients[0].Name
				rpcPackage = g.opts.RPCClients[0].Package
			}

			logicData := map[string]any{
				"Package":      naming.SanitizePackageName(packageName),
				"Module":       g.data.Module,
				"LogicName":    route.LogicName,
				"Method":       route.LogicMethod,
				"RequestType":  route.RequestType,
				"ResponseType": route.ResponseType,
				"HasRequest":   route.HasRequest,
				"HasResponse":  route.HasResponse,
				"HasRPCClient": g.data.HasRPCClients,
				"RPCClient":    rpcClient,
				"RPCPackage":   rpcPackage,
			}

			fileName := naming.FileNameFromHandler(route.Handler) + "_logic.go"
			var path string
			if group.Name != "" {
				path = filepath.Join(r.Opt.OutputDir, "internal", "logic", naming.ToSnakeCase(group.Name), fileName)
			} else {
				path = filepath.Join(r.Opt.OutputDir, "internal", "logic", fileName)
			}

			skipped, err := r.Tpl.RenderSkipExisting(r.Files, "api_logic.tpl", path, logicData)
			if err != nil {
				return err
			}
			if skipped {
				fmt.Printf("    [skip] %s (already exists)\n", filepath.Base(path))
			}
		}
	}
	return nil
}

func (g *APIGenerator) stepRoutes(_ context.Context, r *gen.Runner) error {
	// Generate routes_gen.go (always overwrite)
	if err := r.Tpl.RenderToFile(r.Files, "api_routes_gen.tpl",
		filepath.Join(r.Opt.OutputDir, "internal/handler/routes_gen.go"),
		map[string]any{
			"Module":  g.data.Module,
			"Groups":  g.data.Groups,
			"Imports": g.data.Imports,
		}); err != nil {
		return err
	}

	// Generate routes.go (skip if exists - user editable)
	_, err := r.Tpl.RenderSkipExisting(r.Files, "api_routes.tpl",
		filepath.Join(r.Opt.OutputDir, "internal/handler/routes.go"),
		map[string]any{
			"Module": g.data.Module,
		})
	return err
}

func (g *APIGenerator) stepMain(_ context.Context, r *gen.Runner) error {
	return r.Tpl.RenderToFile(r.Files, "api_main.tpl",
		filepath.Join(r.Opt.OutputDir, "cmd/main.go"),
		map[string]any{
			"Module": g.data.Module,
		})
}

func (g *APIGenerator) stepConfigYAML(_ context.Context, r *gen.Runner) error {
	return r.Tpl.RenderToFile(r.Files, "api_config_yaml.tpl",
		filepath.Join(r.Opt.OutputDir, "etc/api.yaml"),
		map[string]any{
			"ServiceName": g.data.ServiceName,
			"Port":        g.data.Port,
			"RPCClients":  g.data.RPCClients,
			"WithTrace":   g.data.WithTrace,
			"WithDB":      g.data.WithDB,
			"WithRedis":   g.data.WithRedis,
		})
}

func (g *APIGenerator) stepGoMod(_ context.Context, r *gen.Runner) error {
	moduleSet := make(map[string]bool)
	for _, c := range g.opts.RPCClients {
		if c.ModelModule != "" {
			moduleSet[c.ModelModule] = true
		} else {
			moduleSet[c.DMSModule] = true
		}
	}

	var dmsModules []string
	for mod := range moduleSet {
		dmsModules = append(dmsModules, mod)
	}

	return r.Tpl.RenderToFile(r.Files, "api_go_mod.tpl",
		filepath.Join(r.Opt.OutputDir, "go.mod"),
		map[string]any{
			"Module":        g.data.Module,
			"GoVersion":     g.data.GoVersion,
			"HasRPCClients": g.data.HasRPCClients,
			"DMSModules":    dmsModules,
		})
}

// buildDirectories returns the list of directories to create.
func (g *APIGenerator) buildDirectories() []string {
	dirs := []string{
		"cmd",
		"internal/config",
		"internal/handler",
		"internal/logic",
		"internal/middleware",
		"internal/pkg/httputil",
		"internal/svc",
		"internal/types",
		"etc",
	}

	// Add group-specific directories
	for _, group := range g.spec.Groups {
		if group.Annotation != nil && group.Annotation.Group != "" {
			groupName := naming.ToSnakeCase(group.Annotation.Group)
			dirs = append(dirs,
				filepath.Join("internal", "handler", groupName),
				filepath.Join("internal", "logic", groupName),
			)
		}
	}

	return dirs
}

// printGeneratedStructure prints the generated directory structure.
func (g *APIGenerator) printGeneratedStructure() {
	fmt.Println()
	fmt.Println("Code generation completed successfully!")
	fmt.Println()
	fmt.Printf("Generated structure:\n")
	fmt.Printf("  %s/\n", g.opts.Output)
	fmt.Printf("  ├── cmd/\n")
	fmt.Printf("  │   └── main.go\n")
	fmt.Printf("  ├── internal/\n")
	fmt.Printf("  │   ├── config/\n")
	fmt.Printf("  │   │   └── config.go\n")
	fmt.Printf("  │   ├── handler/\n")
	for _, group := range g.spec.Groups {
		if group.Annotation != nil && group.Annotation.Group != "" {
			groupName := naming.ToSnakeCase(group.Annotation.Group)
			fmt.Printf("  │   │   └── %s/\n", groupName)
		}
	}
	fmt.Printf("  │   ├── logic/\n")
	for _, group := range g.spec.Groups {
		if group.Annotation != nil && group.Annotation.Group != "" {
			groupName := naming.ToSnakeCase(group.Annotation.Group)
			fmt.Printf("  │   │   └── %s/\n", groupName)
		}
	}
	fmt.Printf("  │   ├── middleware/\n")
	fmt.Printf("  │   │   └── middleware.go\n")
	fmt.Printf("  │   ├── svc/\n")
	fmt.Printf("  │   │   └── service_context.go\n")
	fmt.Printf("  │   └── types/\n")
	fmt.Printf("  │       └── types.go\n")
	fmt.Printf("  ├── etc/\n")
	fmt.Printf("  │   └── api.yaml\n")
	fmt.Printf("  └── go.mod\n")
}

// Helper functions

// getGroupName extracts the group name from spec.Group.
func getGroupName(group *spec.Group) string {
	if group.Annotation != nil && group.Annotation.Group != "" {
		return group.Annotation.Group
	}
	return ""
}

// getGroupPrefix extracts the prefix from spec.Group.
func getGroupPrefix(group *spec.Group) string {
	if group.Annotation != nil && group.Annotation.Prefix != "" {
		return group.Annotation.Prefix
	}
	return ""
}

// getGroupMiddleware extracts middleware from spec.Group.
func getGroupMiddleware(group *spec.Group) []string {
	if group.Annotation != nil {
		return group.Annotation.Middleware
	}
	return nil
}
