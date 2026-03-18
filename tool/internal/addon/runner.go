package addon

import (
	"fmt"
	"os"
)

// RunOpts configures addon execution behaviour.
type RunOpts struct {
	// DryRun prints what would be done without modifying files.
	DryRun bool
	// Verbose enables verbose logging.
	Verbose bool
	// Quiet suppresses non-essential output.
	Quiet bool
}

// RunAddons executes the requested addons for dir in deterministic registration
// order. The addons slice lists addon names; unknown names return an error.
// Pass an empty slice to run nothing; pass DetectAddons(dir) to auto-run.
func RunAddons(dir string, addons []string, opts RunOpts) error {
	// Resolve to ordered subset that matches registration order.
	toRun := resolveOrdered(addons)

	for _, def := range toRun {
		if err := runAddon(def, dir, opts); err != nil {
			return fmt.Errorf("addon %s: %w", def.Name, err)
		}
	}
	return nil
}

// resolveOrdered filters RegisteredAddons to only those whose names appear in
// requested, preserving registration order.
func resolveOrdered(requested []string) []AddonDef {
	set := make(map[string]bool, len(requested))
	for _, name := range requested {
		set[name] = true
	}
	var out []AddonDef
	for _, a := range RegisteredAddons {
		if set[a.Name] {
			out = append(out, a)
		}
	}
	return out
}

// runAddon executes (or dry-runs) a single addon.
func runAddon(def AddonDef, dir string, opts RunOpts) error {
	if opts.DryRun {
		if !opts.Quiet {
			fmt.Fprintf(os.Stdout, "dry-run: addon %s in %s\n", def.Name, dir)
		}
		return nil
	}
	if opts.Verbose {
		fmt.Fprintf(os.Stdout, "addon: running %s\n", def.Name)
	}
	return def.Run(dir, opts)
}

// runSQLC handles sqlc code generation.
// Ownership: writes to db/ or internal/db/ — does not touch transport files.
func runSQLC(dir string, opts RunOpts) error {
	if opts.Verbose {
		fmt.Fprintf(os.Stdout, "  sqlc: generating from sqlc.yaml in %s\n", dir)
	}
	// TODO: exec `sqlc generate` in dir when binary is available.
	return nil
}

// runRedis generates Redis client stubs.
// Ownership: writes to internal/redis/ — does not overlap with transport files.
func runRedis(dir string, opts RunOpts) error {
	if opts.Verbose {
		fmt.Fprintf(os.Stdout, "  redis: generating client stubs in %s\n", dir)
	}
	// TODO: emit Redis client bootstrap from template.
	return nil
}

// runTracing generates OpenTelemetry tracing bootstrap.
// Ownership: writes to internal/tracing/ — does not overlap with transport files.
func runTracing(dir string, opts RunOpts) error {
	if opts.Verbose {
		fmt.Fprintf(os.Stdout, "  tracing: generating otel bootstrap in %s\n", dir)
	}
	// TODO: emit tracing init from template.
	return nil
}
