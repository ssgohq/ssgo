package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRpcConfig_MissingFile(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadRpcConfig(dir)
	if err != nil {
		t.Fatalf("expected no error for missing .ss.yaml, got %v", err)
	}
	if !cfg.IsEmpty() {
		t.Errorf("expected empty config for missing file, got %+v", cfg)
	}
}

func TestLoadRpcConfig_FullConfig(t *testing.T) {
	dir := t.TempDir()
	yaml := `
rpc:
  proto_module:
    dir: shared-proto
    gen_path: kitex_gen
  services:
    - dir: auth-svc
      protos:
        - proto/auth/v1/auth.proto
      options:
        with_trace: true
        with_redis: false
    - dir: user-svc
      protos:
        - proto/user/v1/user.proto
      options:
        with_trace: true
`
	if err := os.WriteFile(filepath.Join(dir, ".ss.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadRpcConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.IsEmpty() {
		t.Fatal("expected non-empty config")
	}

	pm := cfg.ProtoModuleConfig()
	if pm.Dir != "shared-proto" {
		t.Errorf("ProtoModule.Dir: got %q, want %q", pm.Dir, "shared-proto")
	}
	if pm.GenPath != "kitex_gen" {
		t.Errorf("ProtoModule.GenPath: got %q, want %q", pm.GenPath, "kitex_gen")
	}

	if len(cfg.Services) != 2 {
		t.Fatalf("Services count: got %d, want 2", len(cfg.Services))
	}

	svc0 := cfg.Services[0]
	if svc0.Dir != "auth-svc" {
		t.Errorf("Services[0].Dir: got %q, want %q", svc0.Dir, "auth-svc")
	}
	if len(svc0.Protos) != 1 || svc0.Protos[0] != "proto/auth/v1/auth.proto" {
		t.Errorf("Services[0].Protos: got %v", svc0.Protos)
	}
	if !svc0.Options.WithTrace {
		t.Errorf("Services[0].Options.WithTrace: expected true")
	}
	if svc0.Options.WithRedis {
		t.Errorf("Services[0].Options.WithRedis: expected false")
	}
}

func TestLoadRpcConfig_NoRpcSection(t *testing.T) {
	dir := t.TempDir()
	yaml := `
run:
  build_delay: 500ms
  services:
    - name: my-svc
      cmd: go run ./cmd
`
	if err := os.WriteFile(filepath.Join(dir, ".ss.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadRpcConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.IsEmpty() {
		t.Errorf("expected empty rpc config when rpc: section is absent, got %+v", cfg)
	}
}

func TestLoadRpcConfig_InvalidYaml(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".ss.yaml"), []byte("invalid: [yaml: {"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadRpcConfig(dir)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestProtoModuleConfig_EffectiveGenPath_Default(t *testing.T) {
	m := ProtoModuleConfig{}
	if got := m.EffectiveGenPath(); got != "kitex_gen" {
		t.Errorf("EffectiveGenPath default: got %q, want %q", got, "kitex_gen")
	}
}

func TestProtoModuleConfig_EffectiveGenPath_Custom(t *testing.T) {
	m := ProtoModuleConfig{GenPath: "custom_gen"}
	if got := m.EffectiveGenPath(); got != "custom_gen" {
		t.Errorf("EffectiveGenPath custom: got %q, want %q", got, "custom_gen")
	}
}

func TestRpcConfig_IsEmpty(t *testing.T) {
	tests := []struct {
		name  string
		cfg   RpcConfig
		empty bool
	}{
		{"zero value", RpcConfig{}, true},
		{"proto module dir set", RpcConfig{ProtoModule: ProtoModuleConfig{Dir: "proto-mod"}}, false},
		{"services set", RpcConfig{Services: []ServiceConfig{{Dir: "my-svc"}}}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.cfg.IsEmpty(); got != tc.empty {
				t.Errorf("IsEmpty() = %v, want %v", got, tc.empty)
			}
		})
	}
}
