package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ssgohq/ssgo/internal/util/log"
	"github.com/ssgohq/ssgo/tool/internal/cmdctx"
	gen "github.com/ssgohq/ssgo/tool/internal/generator/kitex"
)

// cloneCtxWithArgs returns a shallow copy of ctx with Args replaced.
func cloneCtxWithArgs(ctx *cmdctx.Context, args []string) *cmdctx.Context {
	return &cmdctx.Context{
		Args:       args,
		Flags:      ctx.Flags,
		WorkingDir: ctx.WorkingDir,
		Debug:      ctx.Debug,
	}
}

// resolveFlag returns the value of the long-form flag, falling back to the
// short-form flag when the long-form flag is absent.
func resolveFlag(ctx *cmdctx.Context, long, short string) string {
	if v := ctx.GetFlag(long); v != "" {
		return v
	}
	return ctx.GetFlag(short)
}

// buildProtoIncludes returns the set of -I include directories that should be
// passed to kitex / protoc based on the absolute path to the .proto file.
func buildProtoIncludes(absProtoFile string) []string {
	includes := []string{}
	protoDir := filepath.Dir(absProtoFile)
	if protoDir == "" || protoDir == "." {
		return includes
	}
	includes = append(includes, protoDir)
	parentDir := filepath.Dir(protoDir)
	if parentDir != "" && parentDir != "." && parentDir != protoDir {
		includes = append(includes, parentDir)
	}
	return includes
}

// genOptions holds the resolved inputs for runGen.
type genOptions struct {
	absProtoFile string
	absOutputDir string
	module       string
	service      string
	useTypes     string
	genPath      string
	withTrace    bool
	withRedis    bool
}

// resolveGenOptions parses and validates all flags / auto-detections required
// by runGen. Splitting this out of runGen reduces its cyclomatic complexity.
func resolveGenOptions(ctx *cmdctx.Context) (*genOptions, error) {
	protoFile := resolveFlag(ctx, "proto", "p")
	if protoFile == "" {
		return nil, nil // caller should print help
	}

	outputDir := resolveFlag(ctx, "dir", "o")
	if outputDir == "" {
		outputDir = "."
	}

	absProtoFile, err := filepath.Abs(protoFile)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute proto path: %w", err)
	}

	proto, err := gen.ParseProto(absProtoFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse proto file for auto-detection: %w", err)
	}

	service, err := resolveService(ctx, proto)
	if err != nil {
		return nil, err
	}

	module, err := resolveModule(ctx, outputDir)
	if err != nil {
		return nil, err
	}

	useTypes := resolveUseTypes(ctx, proto)

	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	return &genOptions{
		absProtoFile: absProtoFile,
		absOutputDir: absOutputDir,
		module:       module,
		service:      service,
		useTypes:     useTypes,
		genPath:      ctx.GetFlag("gen-path"),
		withTrace:    ctx.GetFlagBool("with-trace"),
		withRedis:    ctx.GetFlagBool("with-redis"),
	}, nil
}

// resolveService returns the service name from flags or auto-detects it from
// the parsed proto when exactly one service is defined.
func resolveService(ctx *cmdctx.Context, proto *gen.Proto) (string, error) {
	service := resolveFlag(ctx, "service", "s")
	if service != "" {
		return service, nil
	}
	switch len(proto.Services) {
	case 1:
		log.Info("Auto-detected service from proto: %s", proto.Services[0].Name)
		return proto.Services[0].Name, nil
	case 0:
		return "", fmt.Errorf("no services found in proto file, --service flag is required")
	default:
		names := make([]string, len(proto.Services))
		for i, s := range proto.Services {
			names[i] = s.Name
		}
		return "", fmt.Errorf(
			"multiple services found in proto (%s), please specify --service",
			strings.Join(names, ", "),
		)
	}
}

// resolveModule returns the Go module name from flags or by reading go.mod in
// the output directory.
func resolveModule(ctx *cmdctx.Context, outputDir string) (string, error) {
	module := resolveFlag(ctx, "module", "m")
	if module != "" {
		return module, nil
	}
	detected, err := gen.ReadModuleFromGoMod(outputDir)
	if err != nil {
		return "", fmt.Errorf("failed to read go.mod in %s: %w", outputDir, err)
	}
	if detected != "" {
		log.Info("Auto-detected module from go.mod: %s", detected)
		return detected, nil
	}
	return "", fmt.Errorf("--module or -m flag is required (or place a go.mod in the output directory)")
}

// resolveUseTypes returns the --use import path, either from the flag or
// auto-derived from the proto's go_package option.
func resolveUseTypes(ctx *cmdctx.Context, proto *gen.Proto) string {
	if use := ctx.GetFlag("use"); use != "" {
		return use
	}
	if proto.RawGoPackage == "" {
		return ""
	}
	derived := gen.GoPackageToImportPath(proto.RawGoPackage)
	if derived != "" {
		log.Info("Auto-derived --use from proto go_package: %s", derived)
	}
	return derived
}

// modelOptions holds the resolved inputs for runModel.
type modelOptions struct {
	absProtoFile string
	absOutputDir string
	module       string
	genPath      string
	verbose      bool
}

// resolveModelOptions parses and validates all flags / auto-detections required
// by runModel.
func resolveModelOptions(ctx *cmdctx.Context) (*modelOptions, error) {
	protoFile := resolveFlag(ctx, "proto", "p")
	if protoFile == "" {
		return nil, fmt.Errorf("--proto or -p flag is required")
	}

	outputDir := resolveFlag(ctx, "dir", "o")
	if outputDir == "" {
		return nil, fmt.Errorf("--dir or -o flag is required for model generation")
	}

	module, err := resolveModule(ctx, outputDir)
	if err != nil {
		return nil, err
	}

	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	absProtoFile, err := filepath.Abs(protoFile)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute proto path: %w", err)
	}

	return &modelOptions{
		absProtoFile: absProtoFile,
		absOutputDir: absOutputDir,
		module:       module,
		genPath:      ctx.GetFlag("gen-path"),
		verbose:      ctx.Debug,
	}, nil
}
