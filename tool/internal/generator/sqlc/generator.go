// Package sqlc provides code generation for SQLC-based database layers
package sqlc

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GenOptions represents options for the gen command
type GenOptions struct {
	OutputDir string // Output directory (service root)
	Verbose   bool   // Verbose mode
	WithTrace bool   // Enable OpenTelemetry tracing
}

// GenCommand handles the sqlc gen command
type GenCommand struct {
	opts GenOptions
}

// NewGenCommand creates a new GenCommand
func NewGenCommand(opts GenOptions) *GenCommand {
	return &GenCommand{opts: opts}
}

// Generate runs sqlc generate and updates ServiceContext
func (g *GenCommand) Generate() error {
	fmt.Println("Running SQLC code generation...")

	// Step 1: Run sqlc generate
	runner := NewRunner(g.opts.OutputDir, g.opts.Verbose)
	if err := runner.Generate(); err != nil {
		return err
	}
	fmt.Println("  ✓ SQLC code generation complete")
	fmt.Println()

	// Step 2: Read module from go.mod
	module, err := g.readModule()
	if err != nil {
		return err
	}

	// Step 3: Generate store files
	fmt.Println("Generating store files...")
	storeGen := NewStoreGenerator(StoreGenOptions{
		OutputDir: g.opts.OutputDir,
		Module:    module,
		WithTrace: g.opts.WithTrace,
		Verbose:   g.opts.Verbose,
	})
	if err := storeGen.Generate(); err != nil {
		return fmt.Errorf("failed to generate store: %w", err)
	}
	fmt.Println("  ✓ Store files generated")

	// Step 4: Generate repositories from SQLC output
	fmt.Println("Generating repositories...")
	repoGen := NewRepoGenerator(RepoGenOptions{
		OutputDir: g.opts.OutputDir,
		Module:    module,
		WithTrace: g.opts.WithTrace,
		Verbose:   g.opts.Verbose,
	})
	if err := repoGen.GenerateFromSqlc(); err != nil {
		return fmt.Errorf("failed to generate repositories: %w", err)
	}
	fmt.Println("  ✓ Repositories generated")

	// Step 5: Update config.go with DBConfig
	if g.opts.Verbose {
		fmt.Println("Updating config.go...")
	}
	configUpdater := NewConfigUpdater(g.opts.OutputDir, module, g.opts.Verbose)
	if err := configUpdater.Update(); err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}

	// Step 6: Update etc/config.yaml with db section
	if err := configUpdater.UpdateConfigYaml(); err != nil {
		return fmt.Errorf("failed to update config.yaml: %w", err)
	}

	// Step 7: Update ServiceContext to inject Store
	if g.opts.Verbose {
		fmt.Println("Updating ServiceContext...")
	}
	svcUpdater := NewSvcContextUpdater(g.opts.OutputDir, module, g.opts.Verbose)
	if err := svcUpdater.Update(); err != nil {
		return fmt.Errorf("failed to update ServiceContext: %w", err)
	}
	fmt.Println("  ✓ ServiceContext updated")

	// Step 8: Update main.go to handle NewServiceContext error
	if err := svcUpdater.UpdateMainGo(); err != nil {
		return fmt.Errorf("failed to update main.go: %w", err)
	}

	g.printSuccess()
	return nil
}

// readModule reads the module name from go.mod
func (g *GenCommand) readModule() (string, error) {
	gomodPath := filepath.Join(g.opts.OutputDir, "go.mod")
	data, err := os.ReadFile(gomodPath)
	if err != nil {
		return "", fmt.Errorf("go.mod not found: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimPrefix(line, "module "), nil
		}
	}

	return "", fmt.Errorf("could not parse module from go.mod")
}

// printSuccess prints the success message
func (g *GenCommand) printSuccess() {
	fmt.Println()
	fmt.Println("✓ SQLC code generation complete!")
	fmt.Println()
	fmt.Println("Generated/Updated files:")
	fmt.Println("  internal/data/db/")
	fmt.Println("    - db.go (database interface)")
	fmt.Println("    - models.go (generated models)")
	fmt.Println("    - querier.go (query interface)")
	fmt.Println("    - *.sql.go (query implementations)")
	fmt.Println("  internal/repository/")
	fmt.Println("    - *_repository.go (repository interfaces & implementations)")
	fmt.Println("  internal/store/")
	fmt.Println("    - store.go (store wrapper with transaction support)")
	fmt.Println("    - db.go (pgxpool helper" + func() string {
		if g.opts.WithTrace {
			return " with tracing"
		}
		return ""
	}() + ")")
	fmt.Println("  internal/config/config.go (DBConfig struct)")
	fmt.Println("  internal/svc/service_context.go (Store injected)")
	fmt.Println("  cmd/main.go (updated to handle DB errors)")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Update etc/config.yaml with database settings")
	fmt.Println("  2. Use Store in your logic files:")
	fmt.Println("     user, err := l.svcCtx.Store.Queries().GetUserByID(ctx, id)")
}
