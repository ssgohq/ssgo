package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ssgohq/ssgo/internal/util/log"
	"github.com/ssgohq/ssgo/tool/internal/cmdctx"
)

// runNew creates a new .proto file template
func runNew(ctx *cmdctx.Context) error {
	// Check for help flag
	if ctx.GetFlagBool("help") || ctx.GetFlagBool("h") {
		return printNewHelp()
	}

	if len(ctx.Args) < 1 {
		return printNewHelp()
	}
	serviceName := ctx.Args[0]

	outputDir := ctx.GetFlag("dir")
	if outputDir == "" {
		outputDir = ctx.GetFlag("o")
	}
	if outputDir == "" {
		outputDir = "idl"
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate proto file content
	protoContent := generateProtoTemplate(serviceName)

	// Write proto file
	protoFile := filepath.Join(outputDir, serviceName+".proto")
	if _, err := os.Stat(protoFile); err == nil {
		return fmt.Errorf("proto file already exists: %s", protoFile)
	}

	if err := os.WriteFile(protoFile, []byte(protoContent), 0o644); err != nil {
		return fmt.Errorf("failed to write proto file: %w", err)
	}

	log.Success("Created: %s", protoFile)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Edit the proto file to define your service")
	fmt.Printf("  2. ss rpc gen --proto %s --service %sService -m <module>\n", protoFile, toPascalCase(serviceName))

	return nil
}

func printNewHelp() error {
	fmt.Println(`ss rpc new - Create a new .proto file template

Usage:
	 ss rpc new <service-name> [options]

Arguments:
	 service-name    Name of the service (required)

Options:
	 -o, --dir <dir>   Output directory (default: idl)
	 -h, --help        Show this help

Examples:
	 ss rpc new user
	 ss rpc new product -o proto`)
	return nil
}

func generateProtoTemplate(serviceName string) string {
	pascal := toPascalCase(serviceName)
	lower := strings.ToLower(serviceName)

	return fmt.Sprintf(`syntax = "proto3";

package %[1]s;

option go_package = "%[1]s";

// %[2]sService provides RPC methods for %[1]s operations
service %[2]sService {
  // Get%[2]s retrieves a %[1]s by ID
  rpc Get%[2]s (Get%[2]sRequest) returns (Get%[2]sResponse);

  // Create%[2]s creates a new %[1]s
  rpc Create%[2]s (Create%[2]sRequest) returns (Create%[2]sResponse);

  // Update%[2]s updates an existing %[1]s
  rpc Update%[2]s (Update%[2]sRequest) returns (Update%[2]sResponse);

  // Delete%[2]s removes a %[1]s by ID
  rpc Delete%[2]s (Delete%[2]sRequest) returns (Delete%[2]sResponse);

  // List%[2]ss retrieves a list of %[1]ss
  rpc List%[2]ss (List%[2]ssRequest) returns (List%[2]ssResponse);
}

// %[2]s represents a %[1]s entity
message %[2]s {
  int64 id = 1;
  string name = 2;
  int64 created_at = 3;
  int64 updated_at = 4;
}

message Get%[2]sRequest {
  int64 id = 1;
}

message Get%[2]sResponse {
  %[2]s %[1]s = 1;
}

message Create%[2]sRequest {
  string name = 1;
}

message Create%[2]sResponse {
  %[2]s %[1]s = 1;
}

message Update%[2]sRequest {
  int64 id = 1;
  string name = 2;
}

message Update%[2]sResponse {
  %[2]s %[1]s = 1;
}

message Delete%[2]sRequest {
  int64 id = 1;
}

message Delete%[2]sResponse {
  bool success = 1;
}

message List%[2]ssRequest {
  int32 page = 1;
  int32 page_size = 2;
}

message List%[2]ssResponse {
  repeated %[2]s %[1]ss = 1;
  int32 total = 2;
}
`, lower, pascal)
}

func toPascalCase(s string) string {
	words := strings.FieldsFunc(s, func(c rune) bool {
		return c == '_' || c == '-' || c == ' '
	})

	var result strings.Builder
	for _, word := range words {
		if len(word) > 0 {
			result.WriteString(strings.ToUpper(word[:1]))
			if len(word) > 1 {
				result.WriteString(strings.ToLower(word[1:]))
			}
		}
	}
	return result.String()
}
