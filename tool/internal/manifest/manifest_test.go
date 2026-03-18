package manifest_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ssgohq/ssgo/tool/internal/manifest"
)

func TestValidate_Valid(t *testing.T) {
	m := &manifest.ServiceManifest{
		Binary: manifest.BinaryConfig{Mode: manifest.BinaryModeSplit},
		Transports: manifest.TransportsConfig{
			API: manifest.APITransportConfig{Enabled: true, APIs: []string{"api/user.api"}},
			RPC: manifest.RPCTransportConfig{Enabled: true, Protos: []string{"proto/user.proto"}},
		},
		Addons: []string{"sqlc", "redis"},
	}
	if err := manifest.Validate(m); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidate_SingleModeNoTransport(t *testing.T) {
	m := &manifest.ServiceManifest{
		Binary: manifest.BinaryConfig{Mode: manifest.BinaryModeSingle},
	}
	err := manifest.Validate(m)
	if err == nil {
		t.Fatal("expected validation error for single mode without transports")
	}
}

func TestValidate_APIEnabledNoFiles(t *testing.T) {
	m := &manifest.ServiceManifest{
		Transports: manifest.TransportsConfig{
			API: manifest.APITransportConfig{Enabled: true, APIs: nil},
		},
	}
	err := manifest.Validate(m)
	if err == nil {
		t.Fatal("expected validation error for api enabled but no apis listed")
	}
}

func TestValidate_RPCEnabledNoProtos(t *testing.T) {
	m := &manifest.ServiceManifest{
		Transports: manifest.TransportsConfig{
			RPC: manifest.RPCTransportConfig{Enabled: true, Protos: nil},
		},
	}
	err := manifest.Validate(m)
	if err == nil {
		t.Fatal("expected validation error for rpc enabled but no protos listed")
	}
}

func TestValidate_UnknownAddon(t *testing.T) {
	m := &manifest.ServiceManifest{
		Addons: []string{"unknown-addon"},
	}
	err := manifest.Validate(m)
	if err == nil {
		t.Fatal("expected validation error for unknown addon")
	}
}

func TestValidate_DuplicateAddon(t *testing.T) {
	m := &manifest.ServiceManifest{
		Addons: []string{"sqlc", "sqlc"},
	}
	err := manifest.Validate(m)
	if err == nil {
		t.Fatal("expected validation error for duplicate addon")
	}
}

func TestValidate_InvalidBinaryMode(t *testing.T) {
	m := &manifest.ServiceManifest{
		Binary: manifest.BinaryConfig{Mode: "monolith"},
	}
	err := manifest.Validate(m)
	if err == nil {
		t.Fatal("expected validation error for invalid binary mode")
	}
}

func TestLoadSave_RoundTrip(t *testing.T) {
	dir := t.TempDir()

	// Save a manifest.
	original := &manifest.ServiceManifest{
		Binary: manifest.BinaryConfig{Mode: manifest.BinaryModeSplit},
		Transports: manifest.TransportsConfig{
			API: manifest.APITransportConfig{
				Enabled: true,
				APIs:    []string{"api/user.api"},
				Options: manifest.APITransportOptions{Port: 8080},
			},
			RPC: manifest.RPCTransportConfig{
				Enabled: true,
				Protos:  []string{"proto/user.proto"},
				Options: manifest.RPCTransportOptions{WithTrace: true},
			},
		},
		Addons: []string{"sqlc"},
	}

	if err := manifest.Save(dir, original); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Load it back.
	loaded, err := manifest.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Binary.Mode != original.Binary.Mode {
		t.Errorf("binary.mode mismatch: got %q want %q", loaded.Binary.Mode, original.Binary.Mode)
	}
	if !loaded.Transports.API.Enabled {
		t.Error("expected transports.api.enabled=true")
	}
	if loaded.Transports.API.Options.Port != 8080 {
		t.Errorf("api port: got %d want 8080", loaded.Transports.API.Options.Port)
	}
	if !loaded.Transports.RPC.Enabled {
		t.Error("expected transports.rpc.enabled=true")
	}
	if !loaded.Transports.RPC.Options.WithTrace {
		t.Error("expected rpc.with_trace=true")
	}
	if len(loaded.Addons) != 1 || loaded.Addons[0] != "sqlc" {
		t.Errorf("addons mismatch: got %v", loaded.Addons)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	dir := t.TempDir()
	m, err := manifest.Load(dir)
	if err != nil {
		t.Fatalf("expected nil error for missing .ss.yaml, got: %v", err)
	}
	if m == nil {
		t.Fatal("expected non-nil manifest")
	}
}

func TestSave_PreservesOtherSections(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".ss.yaml")

	// Write an existing .ss.yaml with an rpc section.
	existing := "rpc:\n  proto_module:\n    dir: shared-proto\n"
	if err := os.WriteFile(configPath, []byte(existing), 0o644); err != nil {
		t.Fatalf("write existing: %v", err)
	}

	m := &manifest.ServiceManifest{
		Binary: manifest.BinaryConfig{Mode: manifest.BinaryModeSingle},
		Transports: manifest.TransportsConfig{
			API: manifest.APITransportConfig{Enabled: true, APIs: []string{"api/svc.api"}},
		},
	}
	if err := manifest.Save(dir, m); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, _ := os.ReadFile(configPath)
	content := string(data)
	if !contains(content, "rpc:") {
		t.Error("expected rpc: section to be preserved after Save")
	}
	if !contains(content, "service:") {
		t.Error("expected service: section to be present after Save")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestIsHybrid(t *testing.T) {
	m := &manifest.ServiceManifest{
		Transports: manifest.TransportsConfig{
			API: manifest.APITransportConfig{Enabled: true},
			RPC: manifest.RPCTransportConfig{Enabled: true},
		},
	}
	if !m.IsHybrid() {
		t.Error("expected IsHybrid=true")
	}

	m.Transports.RPC.Enabled = false
	if m.IsHybrid() {
		t.Error("expected IsHybrid=false when RPC disabled")
	}
}
