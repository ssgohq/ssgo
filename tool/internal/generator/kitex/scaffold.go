package gen

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	"github.com/ssgohq/ssgo/internal/util/naming"
	"github.com/ssgohq/ssgo/tool/internal/generator/templates"
)

// ScaffoldOptions represents options for Scaffold
type ScaffoldOptions struct {
	OutputDir    string // Output directory
	Module       string // Go module name
	Service      string // Service name (e.g., UserService)
	ServiceLower string // Lowercase service name (e.g., user)
	Verbose      bool   // Verbose mode
	UseTypes     string // Import path for shared types module (optional)
	ProtoFile    string // Path to proto file (for parsing RPC methods)

	// Optional components for generated code
	WithTrace bool // Enable OpenTelemetry tracing config
	WithRedis bool // Add Redis config
}

// GetTypesModule returns the module path for importing kitex_gen types.
// If UseTypes is specified, extracts the base module. Otherwise uses the local module.
func (o *ScaffoldOptions) GetTypesModule() string {
	if o.UseTypes != "" {
		// UseTypes format: github.com/org/common-pb/kitex_gen/user
		// We need to extract: github.com/org/common-pb
		path := o.UseTypes
		// Remove /kitex_gen/* suffix
		if idx := strings.Index(path, "/kitex_gen/"); idx > 0 {
			return path[:idx]
		}
		return path
	}
	return o.Module
}

// Scaffold generates scaffold files with ServiceContext pattern.
type Scaffold struct {
	opts      ScaffoldOptions
	proto     *Proto // Parsed proto file
	templates *template.Template
}

// NewScaffold creates a new Scaffold generator.
func NewScaffold(opts ScaffoldOptions) *Scaffold {
	funcMap := template.FuncMap{
		"ToSnakeCase":  naming.ToSnakeCase,
		"ToCamelCase":  naming.ToCamelCase,
		"ToPascalCase": naming.ToPascalCase,
		"ToKebabCase":  naming.ToKebabCase,
		"lower":        strings.ToLower,
		"upper":        strings.ToUpper,
		"title":        strings.Title,
	}

	// Parse all templates from embedded filesystem
	tmpl, err := template.New("").Funcs(funcMap).ParseFS(templates.KitexTemplates, "kitex/*.tpl")
	if err != nil {
		// Templates should be valid, panic if not
		panic(fmt.Sprintf("failed to parse templates: %v", err))
	}

	return &Scaffold{
		opts:      opts,
		templates: tmpl,
	}
}

// Generate generates all scaffold files.
func (s *Scaffold) Generate() error {
	// Parse proto file to extract RPC methods
	if s.opts.ProtoFile != "" {
		proto, err := ParseProto(s.opts.ProtoFile)
		if err != nil {
			return fmt.Errorf("failed to parse proto file: %w", err)
		}
		s.proto = proto
		if s.opts.Verbose {
			fmt.Printf("  Parsed proto: %d services, %d total RPCs\n",
				len(proto.Services), s.countTotalRPCs())
		}
	}

	// Build template data
	data := s.buildData()

	// Create directories
	dirs := []string{
		"cmd",
		"internal/config",
		"internal/logic",
		"internal/server",
		"internal/svc",
		"etc",
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(s.opts.OutputDir, dir), 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Generate static files
	staticFiles := []struct {
		tpl          string
		output       string
		skipIfExists bool
	}{
		{"svc.tpl", "internal/svc/service_context.go", true},
		{"config.tpl", "internal/config/config.go", true},
		{"config_yaml.tpl", "etc/config.yaml", true},
		{"main.tpl", "cmd/main.go", true},
		{"go_mod.tpl", "go.mod", false}, // Always regenerate to ensure correct dependencies
	}

	for _, f := range staticFiles {
		outputPath := filepath.Join(s.opts.OutputDir, f.output)
		if f.skipIfExists {
			if _, err := os.Stat(outputPath); err == nil {
				if s.opts.Verbose {
					fmt.Printf("    [skip] %s (already exists)\n", f.output)
				}
				continue
			}
		}

		if err := s.renderToFile(f.tpl, outputPath, data); err != nil {
			return fmt.Errorf("failed to generate %s: %w", f.output, err)
		}
		if s.opts.Verbose {
			fmt.Printf("    [gen] %s\n", f.output)
		}
	}

	// Generate server file (always regenerate)
	serverFile := fmt.Sprintf("internal/server/%s_server.go", naming.ToSnakeCase(data.ServiceLower))
	serverPath := filepath.Join(s.opts.OutputDir, serverFile)
	if err := s.renderToFile("server.tpl", serverPath, data); err != nil {
		return fmt.Errorf("failed to generate server: %w", err)
	}
	if s.opts.Verbose {
		fmt.Printf("    [gen] %s\n", serverFile)
	}

	// Generate logic files (one per method, skip if exists)
	for _, method := range data.Methods {
		methodData := data.WithMethod(&method)
		logicFile := fmt.Sprintf("internal/logic/%s_logic.go", naming.ToSnakeCase(method.Name))
		logicPath := filepath.Join(s.opts.OutputDir, logicFile)

		// Skip if exists
		if _, err := os.Stat(logicPath); err == nil {
			if s.opts.Verbose {
				fmt.Printf("    [skip] %s (already exists)\n", logicFile)
			}
			continue
		}

		if err := s.renderToFile("logic_method.tpl", logicPath, methodData); err != nil {
			return fmt.Errorf("failed to generate logic %s: %w", logicFile, err)
		}
		if s.opts.Verbose {
			fmt.Printf("    [gen] %s\n", logicFile)
		}
	}

	return nil
}

// buildData builds the template data from options and parsed proto.
func (s *Scaffold) buildData() *ScaffoldData {
	methods := s.getMethods()

	// Extract go version from runtime (e.g., "go1.23.0" -> "1.23")
	goVersion := strings.TrimPrefix(runtime.Version(), "go")
	if parts := strings.Split(goVersion, "."); len(parts) >= 2 {
		goVersion = parts[0] + "." + parts[1]
	}

	return &ScaffoldData{
		Module:       s.opts.Module,
		Service:      s.opts.Service,
		ServiceLower: s.opts.ServiceLower,
		TypesModule:  s.opts.GetTypesModule(),
		UseTypes:     s.opts.UseTypes,
		GoVersion:    goVersion,
		WithTrace:    s.opts.WithTrace,
		WithRedis:    s.opts.WithRedis,
		Methods:      methods,
	}
}

// getMethods returns RPC methods from parsed proto or default methods.
func (s *Scaffold) getMethods() []MethodInfo {
	// If proto was parsed, use methods from it
	if s.proto != nil && len(s.proto.Services) > 0 {
		var methods []MethodInfo
		for _, svc := range s.proto.Services {
			for _, rpc := range svc.RPCs {
				methods = append(methods, MethodInfo{
					Name:         rpc.Name,
					RequestType:  rpc.RequestType,
					ResponseType: rpc.ResponseType,
				})
			}
		}
		return methods
	}

	// Fallback: generate default methods based on service name
	serviceName := strings.TrimSuffix(s.opts.Service, "Service")
	return []MethodInfo{
		{
			Name:         fmt.Sprintf("Get%s", serviceName),
			RequestType:  fmt.Sprintf("Get%sRequest", serviceName),
			ResponseType: fmt.Sprintf("Get%sResponse", serviceName),
		},
	}
}

// countTotalRPCs counts total RPC methods across all services.
func (s *Scaffold) countTotalRPCs() int {
	if s.proto == nil {
		return 0
	}
	count := 0
	for _, svc := range s.proto.Services {
		count += len(svc.RPCs)
	}
	return count
}

// renderToFile renders a template to a file.
func (s *Scaffold) renderToFile(tplName, outputPath string, data any) error {
	var buf bytes.Buffer
	if err := s.templates.ExecuteTemplate(&buf, tplName, data); err != nil {
		return fmt.Errorf("failed to execute template %s: %w", tplName, err)
	}

	if err := os.WriteFile(outputPath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", outputPath, err)
	}

	return nil
}
