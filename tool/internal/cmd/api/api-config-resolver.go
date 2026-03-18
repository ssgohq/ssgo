package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/ssgohq/ssgo/internal/util/gomod"
	"github.com/ssgohq/ssgo/tool/internal/cmdctx"
)

// ResolvedApiConfig is the fully resolved runtime config for one API service.
type ResolvedApiConfig struct {
	ApiFile   string
	OutputDir string
	Module    string
	Port      int
	WithLogic bool
	Format    string
}

// ResolveApiConfig merges CLI flags + .ss.yaml config for a single API service.
// Precedence: CLI flag > config value > auto-detect > default.
func ResolveApiConfig(ctx *cmdctx.Context, svc ApiServiceConfig) (*ResolvedApiConfig, error) {
	// API file: required from config (no CLI override for batch mode)
	apiFile := svc.File
	if apiFile == "" {
		return nil, fmt.Errorf("api service config is missing 'file' field")
	}
	if !filepath.IsAbs(apiFile) {
		apiFile = filepath.Join(ctx.WorkingDir, apiFile)
	}

	// Output dir: CLI flag > config value > default "."
	outputDir := ctx.GetFlag("dir")
	if outputDir == "" {
		outputDir = ctx.GetFlag("o")
	}
	if outputDir == "" {
		outputDir = svc.Dir
	}
	if outputDir == "" {
		outputDir = "."
	}
	if !filepath.IsAbs(outputDir) {
		outputDir = filepath.Join(ctx.WorkingDir, outputDir)
	}

	// Module: CLI flag > auto-detect from go.mod in output dir
	module := ctx.GetFlag("module")
	if module == "" {
		module = ctx.GetFlag("m")
	}
	if module == "" {
		module = gomod.ReadModule(outputDir)
	}
	if module == "" {
		return nil, fmt.Errorf("could not detect Go module for %s — add a go.mod or use --module flag", outputDir)
	}

	// Port: CLI flag > config value > default 8080
	port := 8080
	if svc.Options.Port > 0 {
		port = svc.Options.Port
	}

	// WithLogic: CLI flag > config value > default true
	withLogic := true
	if svc.Options.WithLogic != nil {
		withLogic = *svc.Options.WithLogic
	}
	if ctx.HasFlag("with-logic") {
		withLogic = ctx.GetFlagBool("with-logic")
	}

	// Format: CLI flag > config value > default "json"
	format := ctx.GetFlag("format")
	if format == "" {
		format = svc.Options.Format
	}
	if format == "" {
		format = "json"
	}
	if format == "yml" {
		format = "yaml"
	}

	return &ResolvedApiConfig{
		ApiFile:   apiFile,
		OutputDir: outputDir,
		Module:    module,
		Port:      port,
		WithLogic: withLogic,
		Format:    format,
	}, nil
}
