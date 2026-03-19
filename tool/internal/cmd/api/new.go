package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ssgohq/ssgo/internal/util/log"
	"github.com/ssgohq/ssgo/tool/internal/cmdctx"
)

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
		outputDir = "api"
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	filename := filepath.Join(outputDir, serviceName+".api")

	content := generateAPITemplate(serviceName)

	if err := os.WriteFile(filename, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	log.Success("Created %s", filename)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  1. Edit %s to define your API\n", filename)
	fmt.Printf("  2. Run: ss api gen --api %s -m <module-name>\n", filename)

	return nil
}

func printNewHelp() error {
	fmt.Println(`ss api new - Create a new .api file template

Usage:
	 ss api new <service-name> [options]

Arguments:
	 service-name    Name of the service (required)

Options:
	 -o, --dir <dir>   Output directory (default: api)
	 -h, --help        Show this help

Examples:
	 ss api new user
	 ss api new product -o idl`)
	return nil
}

func generateAPITemplate(serviceName string) string {
	return fmt.Sprintf(`syntax = "v1"

info (
    title: "%s API"
    desc: "API service for %s"
    author: "Your Name"
    email: "your@email.com"
    version: "v1.0.0"
)

// Request/Response types
type (
    // GetReq represents a GET request with ID
    GetReq {
        Id int64 `+"`path:\"id\"`"+`
    }

    // ListReq represents a list request with pagination
    ListReq {
        Page     int64 `+"`query:\"page,default=1\"`"+`
        PageSize int64 `+"`query:\"page_size,default=10\"`"+`
    }

    // Item represents a single item
    Item {
        Id        int64  `+"`json:\"id\"`"+`
        Name      string `+"`json:\"name\"`"+`
        CreatedAt string `+"`json:\"created_at\"`"+`
    }

    // ListResp represents a list response
    ListResp {
        Items []Item `+"`json:\"items\"`"+`
        Total int64  `+"`json:\"total\"`"+`
    }

    // CreateReq represents a create request
    CreateReq {
        Name string `+"`json:\"name\" validate:\"required\"`"+`
    }

    // UpdateReq represents an update request
    UpdateReq {
        Id   int64  `+"`path:\"id\"`"+`
        Name string `+"`json:\"name\" validate:\"required\"`"+`
    }

    // DeleteReq represents a delete request
    DeleteReq {
        Id int64 `+"`path:\"id\"`"+`
    }
)

// API routes
@server (
    prefix: /api/v1
    group: %s
)
service %s-api {
    @doc "List all items"
    @handler ListItems
    get /%s (ListReq) returns (ListResp)

    @doc "Get item by ID"
    @handler GetItem
    get /%s/:id (GetReq) returns (Item)

    @doc "Create new item"
    @handler CreateItem
    post /%s (CreateReq) returns (Item)

    @doc "Update item"
    @handler UpdateItem
    put /%s/:id (UpdateReq) returns (Item)

    @doc "Delete item"
    @handler DeleteItem
    delete /%s/:id (DeleteReq)
}
`, serviceName, serviceName, serviceName, serviceName, serviceName, serviceName, serviceName, serviceName, serviceName)
}
