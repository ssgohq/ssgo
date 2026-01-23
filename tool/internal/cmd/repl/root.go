package repl

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ssgohq/ssgo/internal/util/log"
	"github.com/ssgohq/ssgo/tool/internal/cmdctx"
)

// Execute runs the repl command
func Execute(ctx *cmdctx.Context) error {
	// Check evans is installed
	if _, err := exec.LookPath("evans"); err != nil {
		return fmt.Errorf(
			"evans not found. Install with:\n  go install github.com/ktr0731/evans@latest\n  # or: brew install evans",
		)
	}

	// Detect services from .ss.yaml
	services, err := DetectServices(ctx.WorkingDir)
	if err != nil || len(services) == 0 {
		// Fallback: scan current directory for protos
		return runFallback(ctx)
	}

	// Parse args: ss repl [service-name] [evans-args...]
	var selectedService *GrpcService
	var evansArgs []string

	if len(ctx.Args) > 0 {
		// Check if first arg is a service name
		for i := range services {
			if services[i].Name == ctx.Args[0] {
				selectedService = &services[i]
				evansArgs = ctx.Args[1:]
				break
			}
		}
		if selectedService == nil {
			// First arg is not a service name, pass all to evans
			evansArgs = ctx.Args
		}
	}

	// Interactive selection if no service specified
	if selectedService == nil {
		svc, err := SelectService(services)
		if err != nil {
			return err
		}
		selectedService = svc
	}

	// Build and run evans command
	return runEvans(selectedService, evansArgs, ctx.Debug)
}

func runEvans(svc *GrpcService, userArgs []string, debug bool) error {
	var args []string

	// Proto files are in subdirectories like proto/proto/package/*.proto
	// Import statement is "package/file.proto"
	// So we need import path = proto/proto (parent of package dir)

	// Collect all unique parent directories of proto files
	parentDirs := make(map[string]bool)
	for _, proto := range svc.ProtoFiles {
		// Get the directory containing the proto file
		dir := filepath.Dir(proto)
		// Get the parent of that directory (for imports like "package/file.proto")
		parent := filepath.Dir(dir)
		parentDirs[parent] = true
	}

	// Add import paths
	for dir := range parentDirs {
		args = append(args, "--path", dir)
	}

	// Add proto files with relative paths from their parent directories
	addedProtos := make(map[string]bool)
	for _, proto := range svc.ProtoFiles {
		dir := filepath.Dir(proto)
		parent := filepath.Dir(dir)
		rel, _ := filepath.Rel(parent, proto)
		if rel == "" {
			rel = filepath.Base(proto)
		}
		if !addedProtos[rel] {
			addedProtos[rel] = true
			args = append(args, "--proto", rel)
		}
	}

	// Add host and port
	host, port := parseAddress(svc.Address)
	args = append(args, "--host", host)
	args = append(args, "--port", port)

	// Add user args
	args = append(args, userArgs...)

	// Default to REPL mode if no subcommand specified
	hasSubcommand := false
	for _, arg := range userArgs {
		if arg == "repl" || arg == "cli" {
			hasSubcommand = true
			break
		}
	}
	if !hasSubcommand {
		args = append(args, "repl")
	}

	if debug {
		log.Info("Running: evans %s", strings.Join(args, " "))
	}

	cmd := exec.Command("evans", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func runFallback(ctx *cmdctx.Context) error {
	// Scan current directory for protos
	protos, importPaths := DetectProtos(ctx.WorkingDir, nil)
	if len(protos) == 0 {
		return fmt.Errorf("no .proto files found. Create .ss.yaml or place protos in idl/, proto/, api/")
	}

	// Try to detect address from etc/config.yaml
	address, _ := DetectAddress(ctx.WorkingDir)
	if address == "" {
		address = "localhost:8888" // default Kitex port
	}

	svc := &GrpcService{
		Name:        "current",
		Dir:         ctx.WorkingDir,
		ProtoFiles:  protos,
		ImportPaths: importPaths,
		Address:     address,
	}

	return runEvans(svc, ctx.Args, ctx.Debug)
}

func parseAddress(addr string) (string, string) {
	parts := strings.Split(addr, ":")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "localhost", "8888"
}
