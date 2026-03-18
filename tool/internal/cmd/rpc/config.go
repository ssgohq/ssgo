package cmd

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// RpcConfig represents the rpc: section of .ss.yaml.
//
// Example .ss.yaml:
//
//	rpc:
//	  proto_module:
//	    dir: shared-proto
//	    gen_path: kitex_gen
//	  services:
//	    - dir: auth-svc
//	      protos:
//	        - proto/auth/v1/auth.proto
//	      options:
//	        with_trace: true
//	        with_redis: true
type RpcConfig struct {
	ProtoModule ProtoModuleConfig `yaml:"proto_module"`
	Services    []ServiceConfig   `yaml:"services"`
}

// ProtoModuleConfig represents the shared proto module settings.
type ProtoModuleConfig struct {
	// Dir is the directory containing go.mod and proto/ files.
	Dir string `yaml:"dir"`
	// GenPath is the generated code path (default: kitex_gen).
	GenPath string `yaml:"gen_path"`
}

// ServiceConfig represents a single RPC service entry in .ss.yaml.
type ServiceConfig struct {
	// Dir is the service output directory.
	Dir string `yaml:"dir"`
	// Protos is a list of proto file paths relative to ProtoModuleConfig.Dir.
	Protos []string `yaml:"protos"`
	// Options are generation options for this service.
	Options ServiceOptions `yaml:"options"`
}

// ServiceOptions are generation flags for a service.
type ServiceOptions struct {
	WithTrace bool `yaml:"with_trace"`
	WithRedis bool `yaml:"with_redis"`
}

// ssYaml is the full .ss.yaml structure (only rpc section is used here).
type ssYaml struct {
	Rpc RpcConfig `yaml:"rpc"`
}

// LoadRpcConfig reads .ss.yaml from the given directory and returns its rpc section.
// Returns a zero-value RpcConfig (not an error) when .ss.yaml does not exist or
// does not have an rpc: section.
func LoadRpcConfig(workDir string) (*RpcConfig, error) {
	configPath := filepath.Join(workDir, ".ss.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &RpcConfig{}, nil // Optional file — no error
		}
		return nil, err
	}

	var raw ssYaml
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	return &raw.Rpc, nil
}

// GenPath returns the effective gen-path for the proto module (default: "kitex_gen").
func (m *ProtoModuleConfig) EffectiveGenPath() string {
	if m.GenPath != "" {
		return m.GenPath
	}
	return "kitex_gen"
}

// IsEmpty reports whether the RpcConfig has no meaningful configuration.
func (c *RpcConfig) IsEmpty() bool {
	return c.ProtoModuleConfig().Dir == "" && len(c.Services) == 0
}

// ProtoModuleConfig returns the ProtoModule field by value (alias for readability).
func (c *RpcConfig) ProtoModuleConfig() ProtoModuleConfig {
	return c.ProtoModule
}
