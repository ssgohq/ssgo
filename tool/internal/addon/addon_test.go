package addon_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ssgohq/ssgo/tool/internal/addon"
)

// mkfile creates a file with content in dir at relPath.
func mkfile(t *testing.T, dir, rel, content string) {
	t.Helper()
	abs := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkfile MkdirAll: %v", err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatalf("mkfile WriteFile: %v", err)
	}
}

// --- DetectAddons tests ---

func TestDetectAddons_None(t *testing.T) {
	dir := t.TempDir()
	result := addon.DetectAddons(dir)
	if len(result) != 0 {
		t.Errorf("expected no addons, got %v", result)
	}
}

func TestDetectAddons_SQLC_ViaYAML(t *testing.T) {
	dir := t.TempDir()
	mkfile(t, dir, "sqlc.yaml", "version: '2'")

	result := addon.DetectAddons(dir)
	if !contains(result, "sqlc") {
		t.Errorf("expected sqlc in %v", result)
	}
}

func TestDetectAddons_SQLC_ViaSQLFile(t *testing.T) {
	dir := t.TempDir()
	mkfile(t, dir, "db/queries.sql", "SELECT 1;")

	result := addon.DetectAddons(dir)
	if !contains(result, "sqlc") {
		t.Errorf("expected sqlc in %v", result)
	}
}

func TestDetectAddons_Redis(t *testing.T) {
	dir := t.TempDir()
	mkfile(t, dir, "etc/redis.yaml", "addr: localhost:6379")

	result := addon.DetectAddons(dir)
	if !contains(result, "redis") {
		t.Errorf("expected redis in %v", result)
	}
}

func TestDetectAddons_Tracing(t *testing.T) {
	dir := t.TempDir()
	mkfile(t, dir, "etc/otel.yaml", "endpoint: localhost:4317")

	result := addon.DetectAddons(dir)
	if !contains(result, "tracing") {
		t.Errorf("expected tracing in %v", result)
	}
}

func TestDetectAddons_Order_DeterministicRegistrationOrder(t *testing.T) {
	dir := t.TempDir()
	// Set up all three markers.
	mkfile(t, dir, "sqlc.yaml", "")
	mkfile(t, dir, "etc/redis.yaml", "")
	mkfile(t, dir, "etc/otel.yaml", "")

	result := addon.DetectAddons(dir)
	if len(result) != 3 {
		t.Fatalf("expected 3 addons, got %v", result)
	}
	// Must be in registration order: sqlc, redis, tracing.
	if result[0] != "sqlc" || result[1] != "redis" || result[2] != "tracing" {
		t.Errorf("order: got %v want [sqlc redis tracing]", result)
	}
}

// --- RunAddons tests ---

func TestRunAddons_DryRun_NoError(t *testing.T) {
	dir := t.TempDir()
	err := addon.RunAddons(dir, []string{"sqlc", "redis", "tracing"}, addon.RunOpts{DryRun: true})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunAddons_EmptyList(t *testing.T) {
	dir := t.TempDir()
	err := addon.RunAddons(dir, nil, addon.RunOpts{})
	if err != nil {
		t.Errorf("unexpected error for empty addon list: %v", err)
	}
}

func TestRunAddons_UnknownAddon_Skipped(t *testing.T) {
	// Unknown addons are simply not in registered list, so resolveOrdered skips them.
	dir := t.TempDir()
	err := addon.RunAddons(dir, []string{"nonexistent"}, addon.RunOpts{DryRun: true})
	if err != nil {
		t.Errorf("unexpected error for unknown addon: %v", err)
	}
}

func TestKnownAddonNames(t *testing.T) {
	names := addon.KnownAddonNames()
	for _, want := range []string{"sqlc", "redis", "tracing"} {
		if !contains(names, want) {
			t.Errorf("expected %q in KnownAddonNames, got %v", want, names)
		}
	}
}

// contains is a helper to check slice membership.
func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
