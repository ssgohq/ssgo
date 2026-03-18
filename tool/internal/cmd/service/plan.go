package service

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/ssgohq/ssgo/tool/internal/manifest"
	"github.com/ssgohq/ssgo/tool/internal/scanner"
)

// GenerationStep describes one unit of planned work.
type GenerationStep struct {
	Transport string   `json:"transport"`
	Action    string   `json:"action"`
	Contracts []string `json:"contracts"`
}

// Plan is the deterministic output of the plan command.
type Plan struct {
	Root      string           `json:"root"`
	State     string           `json:"state"`
	Steps     []GenerationStep `json:"steps"`
	Conflicts []string         `json:"conflicts,omitempty"`
}

// runPlan implements `ss service plan <dir>` (always a dry-run).
func runPlan(opts Options) error {
	dir := opts.resolveDir()

	result, err := scanner.ScanDir(dir)
	if err != nil {
		return fmt.Errorf("plan: %w", err)
	}

	m, err := manifest.Load(dir)
	if err != nil {
		return fmt.Errorf("plan: load manifest: %w", err)
	}

	plan := buildPlan(result, m)

	if opts.JSON {
		return printJSON(plan)
	}

	if !opts.Quiet {
		fmt.Fprintf(os.Stdout, "Directory : %s\n", plan.Root)
		fmt.Fprintf(os.Stdout, "State     : %s\n", plan.State)
	}

	if len(plan.Steps) == 0 {
		fmt.Fprintln(os.Stdout, "Nothing to generate.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "#\tTRANSPORT\tACTION\tCONTRACTS")
	for i, step := range plan.Steps {
		fmt.Fprintf(w, "%d\t%s\t%s\t%v\n", i+1, step.Transport, step.Action, step.Contracts)
	}
	w.Flush()

	if len(plan.Conflicts) > 0 {
		fmt.Fprintln(os.Stdout, "\nConflicts/Warnings:")
		for _, cf := range plan.Conflicts {
			fmt.Fprintf(os.Stdout, "  %s\n", cf)
		}
	}

	return nil
}

// buildPlan creates a deterministic Plan from scan results and the manifest.
func buildPlan(result *scanner.ScanResult, m *manifest.ServiceManifest) Plan {
	plan := Plan{
		Root:  result.Root,
		State: string(result.State),
	}

	for _, cf := range result.Conflicts {
		plan.Conflicts = append(plan.Conflicts, fmt.Sprintf("[%s] %s", cf.Kind, cf.Message))
	}

	// Follow suggested order for determinism.
	for _, transport := range result.SuggestedOrder {
		switch transport {
		case "rpc":
			protos := contractPaths(result.RPCContracts())
			if len(protos) == 0 {
				continue
			}
			action := "generate"
			if result.State == scanner.ServiceStateRPCOnly || result.State == scanner.ServiceStateHybrid {
				action = "sync"
			}
			// Honour manifest override.
			if m != nil && m.Transports.RPC.Enabled && len(m.Transports.RPC.Protos) > 0 {
				protos = m.Transports.RPC.Protos
			}
			plan.Steps = append(plan.Steps, GenerationStep{
				Transport: "rpc",
				Action:    action,
				Contracts: protos,
			})

		case "api":
			apis := contractPaths(result.APIContracts())
			if len(apis) == 0 {
				continue
			}
			action := "generate"
			if result.State == scanner.ServiceStateAPIOnly || result.State == scanner.ServiceStateHybrid {
				action = "sync"
			}
			if m != nil && m.Transports.API.Enabled && len(m.Transports.API.APIs) > 0 {
				apis = m.Transports.API.APIs
			}
			plan.Steps = append(plan.Steps, GenerationStep{
				Transport: "api",
				Action:    action,
				Contracts: apis,
			})

		case "sqlc":
			sqls := contractPaths(result.SQLContracts())
			if len(sqls) == 0 {
				continue
			}
			plan.Steps = append(plan.Steps, GenerationStep{
				Transport: "sqlc",
				Action:    "generate",
				Contracts: sqls,
			})
		}
	}

	return plan
}

func contractPaths(contracts []scanner.Contract) []string {
	paths := make([]string, 0, len(contracts))
	for _, c := range contracts {
		paths = append(paths, c.Path)
	}
	return paths
}
