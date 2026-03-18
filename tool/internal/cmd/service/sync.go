package service

import (
	"fmt"
	"os"

	"github.com/ssgohq/ssgo/tool/internal/manifest"
	"github.com/ssgohq/ssgo/tool/internal/scanner"
)

// runSync implements `ss service sync <dir>`.
// It regenerates from the manifest with ownership-aware tidy steps:
// previously generated files are preserved unless explicitly overwritten
// by the current manifest.
func runSync(opts Options) error {
	dir := opts.resolveDir()

	m, err := manifest.Load(dir)
	if err != nil {
		return fmt.Errorf("sync: load manifest: %w", err)
	}

	if err := manifest.Validate(m); err != nil {
		return fmt.Errorf("sync: invalid manifest: %w", err)
	}

	result, err := scanner.ScanDir(dir)
	if err != nil {
		return fmt.Errorf("sync: scan: %w", err)
	}

	plan := buildSyncPlan(result, m)

	if len(plan.Steps) == 0 {
		if !opts.Quiet {
			fmt.Fprintln(os.Stdout, "Nothing to sync.")
		}
		return nil
	}

	if opts.JSON {
		return printJSON(map[string]any{
			"root":  dir,
			"steps": plan.Steps,
		})
	}

	for _, step := range plan.Steps {
		if err := executeStep(step, dir, opts); err != nil {
			return fmt.Errorf("sync: step %s/%s: %w", step.Transport, step.Action, err)
		}
	}

	return nil
}

// buildSyncPlan constructs a sync plan driven by the manifest rather than
// raw scan discovery. Ownership policy: only transports listed in the manifest
// are regenerated; unlisted transports are preserved as-is.
func buildSyncPlan(result *scanner.ScanResult, m *manifest.ServiceManifest) Plan {
	plan := Plan{
		Root:  result.Root,
		State: string(result.State),
	}

	// RPC first for dependency order.
	if m.Transports.RPC.Enabled {
		protos := m.Transports.RPC.Protos
		if len(protos) == 0 {
			protos = contractPaths(result.RPCContracts())
		}
		if len(protos) > 0 {
			plan.Steps = append(plan.Steps, GenerationStep{
				Transport: "rpc",
				Action:    "sync",
				Contracts: protos,
			})
		}
	}

	if m.Transports.API.Enabled {
		apis := m.Transports.API.APIs
		if len(apis) == 0 {
			apis = contractPaths(result.APIContracts())
		}
		if len(apis) > 0 {
			plan.Steps = append(plan.Steps, GenerationStep{
				Transport: "api",
				Action:    "sync",
				Contracts: apis,
			})
		}
	}

	// Addons.
	for _, addon := range m.Addons {
		if addon == "sqlc" {
			sqls := contractPaths(result.SQLContracts())
			if len(sqls) > 0 {
				plan.Steps = append(plan.Steps, GenerationStep{
					Transport: "sqlc",
					Action:    "sync",
					Contracts: sqls,
				})
			}
		}
	}

	return plan
}
