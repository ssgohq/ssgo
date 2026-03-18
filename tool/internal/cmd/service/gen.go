package service

import (
	"fmt"
	"os"

	"github.com/ssgohq/ssgo/tool/internal/addon"
	"github.com/ssgohq/ssgo/tool/internal/manifest"
	"github.com/ssgohq/ssgo/tool/internal/scanner"
)

// runGen implements `ss service gen <dir>`.
// It builds a plan and executes each step (or prints what it would do in dry-run).
func runGen(opts Options) error {
	dir := opts.resolveDir()

	result, err := scanner.ScanDir(dir)
	if err != nil {
		return fmt.Errorf("gen: scan: %w", err)
	}

	m, err := manifest.Load(dir)
	if err != nil {
		return fmt.Errorf("gen: load manifest: %w", err)
	}

	if err := manifest.Validate(m); err != nil {
		// Non-fatal: warn but continue with scanned contracts.
		if !opts.Quiet {
			fmt.Fprintf(os.Stderr, "Warning: manifest validation: %v\n", err)
		}
	}

	// When the manifest has no transports configured and we are in an
	// interactive session, prompt the user for setup decisions.
	if !opts.Quiet && !opts.NonInteractive && !m.Transports.API.Enabled && !m.Transports.RPC.Enabled {
		interactive, iErr := InteractiveSetup(dir, result, opts.NonInteractive)
		if iErr != nil {
			return fmt.Errorf("gen: interactive setup: %w", iErr)
		}
		// Merge: prefer prompted values when manifest was empty.
		if interactive.Binary.Mode != "" {
			m.Binary.Mode = interactive.Binary.Mode
		}
		if interactive.Transports.API.Enabled || interactive.Transports.RPC.Enabled {
			m.Transports.API.Enabled = interactive.Transports.API.Enabled
			m.Transports.RPC.Enabled = interactive.Transports.RPC.Enabled
		}
	}

	plan := buildPlan(result, m)

	if len(plan.Steps) == 0 {
		if !opts.Quiet {
			fmt.Fprintln(os.Stdout, "Nothing to generate.")
		}
		return nil
	}

	if len(plan.Conflicts) > 0 {
		PrintConflicts(plan.Conflicts)
	}

	for _, step := range plan.Steps {
		if err := executeStep(step, dir, opts); err != nil {
			return fmt.Errorf("gen: step %s/%s: %w", step.Transport, step.Action, err)
		}
	}

	// Run addon pipeline after transport generation.
	// Resolve addons: prefer manifest list, fall back to auto-detection.
	addonNames := m.Addons
	if len(addonNames) == 0 {
		addonNames = addon.DetectAddons(dir)
	}
	if len(addonNames) > 0 {
		addonOpts := addon.RunOpts{
			DryRun:  opts.DryRun,
			Verbose: opts.Verbose,
			Quiet:   opts.Quiet,
		}
		if err := addon.RunAddons(dir, addonNames, addonOpts); err != nil {
			return fmt.Errorf("gen: addons: %w", err)
		}
	}

	if opts.JSON {
		return printJSON(map[string]any{
			"root":    dir,
			"applied": plan.Steps,
			"addons":  addonNames,
		})
	}
	return nil
}

// executeStep runs (or simulates) a single generation step.
func executeStep(step GenerationStep, dir string, opts Options) error {
	label := fmt.Sprintf("[%s] %s %v", step.Transport, step.Action, step.Contracts)

	if opts.DryRun {
		fmt.Fprintf(os.Stdout, "dry-run: %s\n", label)
		return nil
	}

	if opts.Verbose {
		fmt.Fprintf(os.Stdout, "executing: %s\n", label)
	}

	switch step.Transport {
	case "rpc":
		return genRPC(dir, step, opts)
	case "api":
		return genAPI(dir, step, opts)
	case "sqlc":
		return genSQLC(dir, step, opts)
	default:
		return fmt.Errorf("unknown transport %q", step.Transport)
	}
}

// genRPC delegates to the existing rpc command machinery.
// For now it prints a placeholder; the actual delegation hook is wired in root.go.
func genRPC(_ string, step GenerationStep, opts Options) error {
	if opts.Verbose {
		fmt.Fprintf(os.Stdout, "  rpc: generating from protos: %v\n", step.Contracts)
	}
	// TODO: delegate to rpccmd.Execute with synthesised args.
	return nil
}

// genAPI delegates to the existing api command machinery.
func genAPI(_ string, step GenerationStep, opts Options) error {
	if opts.Verbose {
		fmt.Fprintf(os.Stdout, "  api: generating from apis: %v\n", step.Contracts)
	}
	// TODO: delegate to apicmd.Execute with synthesised args.
	return nil
}

// genSQLC delegates to the addon runner's sqlc addon for schema generation.
func genSQLC(dir string, step GenerationStep, opts Options) error {
	if opts.Verbose {
		fmt.Fprintf(os.Stdout, "  sqlc: generating from schemas: %v\n", step.Contracts)
	}
	return addon.RunAddons(dir, []string{"sqlc"}, addon.RunOpts{
		DryRun:  opts.DryRun,
		Verbose: opts.Verbose,
		Quiet:   opts.Quiet,
	})
}
