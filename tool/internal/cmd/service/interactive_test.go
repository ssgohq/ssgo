package service

import (
	"testing"

	"github.com/ssgohq/ssgo/tool/internal/manifest"
	"github.com/ssgohq/ssgo/tool/internal/scanner"
)

// TestInteractiveSetup_NonInteractiveReturnsEmpty verifies that when
// NonInteractive mode is requested, InteractiveSetup returns a zero manifest
// without prompting or returning an error.
func TestInteractiveSetup_NonInteractiveReturnsEmpty(t *testing.T) {
	dir := t.TempDir()

	result, err := scanner.ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir: %v", err)
	}

	m, err := InteractiveSetup(dir, result, true /* nonInteractive */)
	if err != nil {
		t.Fatalf("InteractiveSetup(nonInteractive=true) returned error: %v", err)
	}
	if m == nil {
		t.Fatal("expected non-nil manifest")
	}
	// Non-interactive mode must not pre-configure any transports.
	if m.Transports.API.Enabled || m.Transports.RPC.Enabled {
		t.Error("expected no transports enabled in non-interactive empty result")
	}
}

// TestPrintPlanSummary_NoPanic verifies that PrintPlanSummary does not panic
// for various manifest and scan states.
func TestPrintPlanSummary_NoPanic(t *testing.T) {
	dir := t.TempDir()

	result, err := scanner.ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir: %v", err)
	}

	cases := []*manifest.ServiceManifest{
		{},
		{Binary: manifest.BinaryConfig{Mode: manifest.BinaryModeSplit}},
		{Binary: manifest.BinaryConfig{Mode: manifest.BinaryModeSingle},
			Transports: manifest.TransportsConfig{
				API: manifest.APITransportConfig{Enabled: true},
				RPC: manifest.RPCTransportConfig{Enabled: true},
			}},
	}

	for _, m := range cases {
		// Must not panic.
		PrintPlanSummary(result, m)
	}
}

// TestPrintConflicts_NoPanic verifies that PrintConflicts handles nil/empty/non-empty slices.
func TestPrintConflicts_NoPanic(t *testing.T) {
	PrintConflicts(nil)
	PrintConflicts([]string{})
	PrintConflicts([]string{"conflict A", "conflict B"})
}
