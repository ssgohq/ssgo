package service

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/ssgohq/ssgo/tool/internal/scanner"
)

// runInspect implements `ss service inspect <dir>`.
// It scans the directory and prints discovered contracts and current state.
func runInspect(opts Options) error {
	dir := opts.resolveDir()

	result, err := scanner.ScanDir(dir)
	if err != nil {
		return fmt.Errorf("inspect: %w", err)
	}

	if opts.JSON {
		return printJSON(map[string]any{
			"root":            result.Root,
			"state":           result.State,
			"suggested_order": result.SuggestedOrder,
			"contracts":       result.Contracts,
			"conflicts":       result.Conflicts,
		})
	}

	if !opts.Quiet {
		fmt.Fprintf(os.Stdout, "Directory : %s\n", result.Root)
		fmt.Fprintf(os.Stdout, "State     : %s\n", result.State)
	}

	if len(result.Contracts) == 0 {
		if !opts.Quiet {
			fmt.Fprintln(os.Stdout, "No contracts found.")
		}
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "KIND\tPATH")
	for _, c := range result.Contracts {
		fmt.Fprintf(w, "%s\t%s\n", c.Kind, c.Path)
	}
	w.Flush()

	if len(result.Conflicts) > 0 {
		fmt.Fprintln(os.Stdout, "\nConflicts/Warnings:")
		for _, cf := range result.Conflicts {
			fmt.Fprintf(os.Stdout, "  [%s] %s\n", cf.Kind, cf.Message)
		}
	}

	if opts.Verbose && len(result.SuggestedOrder) > 0 {
		fmt.Fprintf(os.Stdout, "\nSuggested generation order: %v\n", result.SuggestedOrder)
	}

	return nil
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
