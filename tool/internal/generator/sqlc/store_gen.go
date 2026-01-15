// Package sqlc provides code generation for SQLC-based database layers
package sqlc

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/ssgohq/ssgo/tool/internal/generator/common"
	"github.com/ssgohq/ssgo/tool/internal/generator/templates"
)

// StoreGenOptions represents options for store generation
type StoreGenOptions struct {
	OutputDir string // Output directory (project root)
	Module    string // Go module name
	WithTrace bool   // Enable OpenTelemetry tracing
	Verbose   bool   // Verbose mode
}

// StoreGenerator generates store layer files
type StoreGenerator struct {
	opts    StoreGenOptions
	funcMap template.FuncMap
}

// NewStoreGenerator creates a new StoreGenerator
func NewStoreGenerator(opts StoreGenOptions) *StoreGenerator {
	g := &StoreGenerator{opts: opts}
	g.funcMap = g.createFuncMap()
	return g
}

// createFuncMap creates template function map
func (g *StoreGenerator) createFuncMap() template.FuncMap {
	return template.FuncMap{
		"ToSnakeCase":  common.ToSnakeCase,
		"ToCamelCase":  common.ToCamelCase,
		"ToPascalCase": common.ToPascalCase,
		"ToKebabCase":  common.ToKebabCase,
		"lower":        strings.ToLower,
		"upper":        strings.ToUpper,
	}
}

// Generate generates store layer files
func (g *StoreGenerator) Generate() error {
	// Create store directory
	storeDir := filepath.Join(g.opts.OutputDir, "internal", "store")
	if err := os.MkdirAll(storeDir, 0o755); err != nil {
		return fmt.Errorf("failed to create store directory: %w", err)
	}

	// Generate store.go
	if err := g.generateStoreGo(); err != nil {
		return fmt.Errorf("failed to generate store.go: %w", err)
	}

	// Generate db.go
	if err := g.generateDBGo(); err != nil {
		return fmt.Errorf("failed to generate db.go: %w", err)
	}

	return nil
}

// generateStoreGo generates internal/store/store.go
func (g *StoreGenerator) generateStoreGo() error {
	data := struct {
		Module string
	}{
		Module: g.opts.Module,
	}

	content, err := g.executeTemplate("sqlc/store.go.tpl", data)
	if err != nil {
		return err
	}

	path := filepath.Join(g.opts.OutputDir, "internal", "store", "store.go")

	// Don't overwrite existing
	if _, err := os.Stat(path); err == nil {
		if g.opts.Verbose {
			fmt.Printf("  %s already exists, skipping...\n", path)
		}
		return nil
	}

	if g.opts.Verbose {
		fmt.Printf("  Creating %s\n", path)
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// generateDBGo generates internal/store/db.go
func (g *StoreGenerator) generateDBGo() error {
	data := struct {
		Module    string
		WithTrace bool
	}{
		Module:    g.opts.Module,
		WithTrace: g.opts.WithTrace,
	}

	content, err := g.executeTemplate("sqlc/db.go.tpl", data)
	if err != nil {
		return err
	}

	path := filepath.Join(g.opts.OutputDir, "internal", "store", "db.go")

	// Don't overwrite existing
	if _, err := os.Stat(path); err == nil {
		if g.opts.Verbose {
			fmt.Printf("  %s already exists, skipping...\n", path)
		}
		return nil
	}

	if g.opts.Verbose {
		fmt.Printf("  Creating %s\n", path)
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// executeTemplate executes a template and returns the result
func (g *StoreGenerator) executeTemplate(tplPath string, data interface{}) (string, error) {
	content, err := templates.SQLCTemplates.ReadFile(tplPath)
	if err != nil {
		return "", fmt.Errorf("failed to read template %s: %w", tplPath, err)
	}

	tpl, err := template.New(filepath.Base(tplPath)).Funcs(g.funcMap).Parse(string(content))
	if err != nil {
		return "", fmt.Errorf("failed to parse template %s: %w", tplPath, err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", tplPath, err)
	}

	return buf.String(), nil
}
