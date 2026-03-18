package composer

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/ssgohq/ssgo/tool/internal/generator/service"
	"github.com/ssgohq/ssgo/tool/internal/generator/templates"
)

// ComposerGenerator generates cmd/server/main.go — a single-binary entrypoint
// that runs API and/or RPC transports in the same process.
type ComposerGenerator struct {
	opts Options
	tmpl *template.Template
}

// New creates a ComposerGenerator with the given options.
func New(opts Options) (*ComposerGenerator, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}
	tmpl, err := template.New("").ParseFS(templates.ComposerTemplates, "composer/*.tpl")
	if err != nil {
		return nil, fmt.Errorf("composer: parse templates: %w", err)
	}
	return &ComposerGenerator{opts: opts, tmpl: tmpl}, nil
}

// Generate runs the shared scaffold (base files) and then produces
// cmd/server/main.go in opts.OutputDir.
func (g *ComposerGenerator) Generate() error {
	if g.opts.Verbose {
		fmt.Printf("Generating single-binary composer entrypoint...\n")
		fmt.Printf("  Output:  %s\n", g.opts.OutputDir)
		fmt.Printf("  Module:  %s\n", g.opts.Module)
		fmt.Printf("  WithAPI: %v  WithRPC: %v\n", g.opts.WithAPI, g.opts.WithRPC)
	}

	// Ensure shared base files (internal/config/base.go, internal/svc/base.go).
	sc := service.NewSharedScaffold(service.ScaffoldOptions{
		OutputDir: g.opts.OutputDir,
		Module:    g.opts.Module,
		Verbose:   g.opts.Verbose,
		SkipGoMod: true,
	})
	if err := sc.Generate(); err != nil {
		return fmt.Errorf("composer: shared scaffold: %w", err)
	}

	// Ensure cmd/server directory exists.
	serverDir := filepath.Join(g.opts.OutputDir, "cmd", "server")
	if err := os.MkdirAll(serverDir, 0o755); err != nil {
		return fmt.Errorf("composer: create cmd/server: %w", err)
	}

	// Render cmd/server/main.go (always overwrite — fully generated).
	mainPath := filepath.Join(serverDir, "main.go")
	data := buildTemplateData(g.opts)
	if err := g.renderToFile("server_main.tpl", mainPath, data); err != nil {
		return fmt.Errorf("composer: generate cmd/server/main.go: %w", err)
	}

	if g.opts.Verbose {
		fmt.Printf("    [gen]  cmd/server/main.go\n")
	}
	fmt.Println("Single-binary entrypoint generated: cmd/server/main.go")
	return nil
}

// renderToFile executes a named template and writes the result to outputPath.
func (g *ComposerGenerator) renderToFile(tplName, outputPath string, data any) error {
	var buf bytes.Buffer
	if err := g.tmpl.ExecuteTemplate(&buf, tplName, data); err != nil {
		return fmt.Errorf("execute template %s: %w", tplName, err)
	}
	if err := os.WriteFile(outputPath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", outputPath, err)
	}
	return nil
}

// errorf returns a formatted error prefixed with "composer: ".
func errorf(format string, args ...any) error {
	return fmt.Errorf("composer: "+format, args...)
}
