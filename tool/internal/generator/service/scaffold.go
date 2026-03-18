// Package service provides shared scaffolding for hybrid API/RPC services.
// It generates transport-neutral base files (internal/config/base.go,
// internal/svc/base.go, go.mod) that both the API and RPC generators depend on.
package service

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
)

// ScaffoldOptions configures SharedScaffold generation.
type ScaffoldOptions struct {
	// OutputDir is the root directory of the service (where go.mod lives).
	OutputDir string

	// Module is the Go module name (e.g., "github.com/org/my-service").
	Module string

	// Verbose enables verbose logging.
	Verbose bool

	// SkipGoMod skips regenerating go.mod (useful when it already exists).
	SkipGoMod bool
}

// Validate returns an error if required options are missing.
func (o ScaffoldOptions) Validate() error {
	if o.OutputDir == "" {
		return fmt.Errorf("service scaffold: OutputDir is required")
	}
	if o.Module == "" {
		return fmt.Errorf("service scaffold: Module is required")
	}
	return nil
}

// scaffoldData is the template data passed to all service templates.
type scaffoldData struct {
	Module    string
	GoVersion string
}

// SharedScaffold generates the transport-neutral base files that both
// API and RPC generators share. It is idempotent: files are written
// only if they do not already exist (skip-if-exists policy).
type SharedScaffold struct {
	opts ScaffoldOptions
	tmpl *template.Template
}

// NewSharedScaffold creates a SharedScaffold with the given options.
func NewSharedScaffold(opts ScaffoldOptions) *SharedScaffold {
	return &SharedScaffold{
		opts: opts,
		tmpl: newTemplates(),
	}
}

// Generate runs the shared scaffold generation.
// Call this before any transport-specific generator so base files are in place.
func (s *SharedScaffold) Generate() error {
	if err := s.opts.Validate(); err != nil {
		return err
	}

	data := s.buildData()

	// Ensure required directories exist.
	dirs := []string{
		"internal/config",
		"internal/svc",
	}
	for _, dir := range dirs {
		full := filepath.Join(s.opts.OutputDir, dir)
		if err := os.MkdirAll(full, 0o755); err != nil {
			return fmt.Errorf("service scaffold: create dir %s: %w", dir, err)
		}
	}

	// Files that are always skip-if-exists (user-editable shared files).
	skipFiles := []struct {
		tpl    string
		output string
	}{
		{"base_config.tpl", "internal/config/base.go"},
		{"base_svc.tpl", "internal/svc/base.go"},
	}

	for _, f := range skipFiles {
		outPath := filepath.Join(s.opts.OutputDir, f.output)
		if _, err := os.Stat(outPath); err == nil {
			if s.opts.Verbose {
				fmt.Printf("    [skip] %s (already exists)\n", f.output)
			}
			continue
		}
		if err := s.renderToFile(f.tpl, outPath, data); err != nil {
			return fmt.Errorf("service scaffold: generate %s: %w", f.output, err)
		}
		if s.opts.Verbose {
			fmt.Printf("    [gen]  %s\n", f.output)
		}
	}

	// go.mod: skip if exists OR if SkipGoMod is set.
	if !s.opts.SkipGoMod {
		goModPath := filepath.Join(s.opts.OutputDir, "go.mod")
		if _, err := os.Stat(goModPath); os.IsNotExist(err) {
			if err := s.renderToFile("go_mod.tpl", goModPath, data); err != nil {
				return fmt.Errorf("service scaffold: generate go.mod: %w", err)
			}
			if s.opts.Verbose {
				fmt.Printf("    [gen]  go.mod\n")
			}
		} else if s.opts.Verbose {
			fmt.Printf("    [skip] go.mod (already exists)\n")
		}
	}

	return nil
}

// buildData constructs template data from scaffold options.
func (s *SharedScaffold) buildData() scaffoldData {
	goVersion := strings.TrimPrefix(runtime.Version(), "go")
	if parts := strings.Split(goVersion, "."); len(parts) >= 2 {
		goVersion = parts[0] + "." + parts[1]
	}
	return scaffoldData{
		Module:    s.opts.Module,
		GoVersion: goVersion,
	}
}

// renderToFile renders the named template to outputPath.
func (s *SharedScaffold) renderToFile(tplName, outputPath string, data any) error {
	var buf bytes.Buffer
	if err := s.tmpl.ExecuteTemplate(&buf, tplName, data); err != nil {
		return fmt.Errorf("render template %s: %w", tplName, err)
	}
	if err := os.WriteFile(outputPath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write file %s: %w", outputPath, err)
	}
	return nil
}
