// Package manifest defines the service manifest schema for .ss.yaml (service: section).
// It supports hybrid API/RPC service generation with configurable binary modes.
package manifest

// BinaryMode controls how service binaries are compiled and deployed.
type BinaryMode string

const (
	// BinaryModeSplit generates separate api/rpc entry points under cmd/api and cmd/rpc.
	BinaryModeSplit BinaryMode = "split"
	// BinaryModeSingle generates a single binary that hosts both transports.
	BinaryModeSingle BinaryMode = "single"
)

// BinaryCommand selects which transport(s) the single-mode binary runs.
type BinaryCommand string

const (
	BinaryCommandAPI     BinaryCommand = "api"
	BinaryCommandRPC     BinaryCommand = "rpc"
	BinaryCommandService BinaryCommand = "service"
	BinaryCommandBoth    BinaryCommand = "both"
)

// BinaryConfig controls binary generation strategy.
type BinaryConfig struct {
	// Mode is either "split" (default) or "single".
	Mode BinaryMode `yaml:"mode"`
	// Command selects which transport a single binary starts (single mode only).
	Command BinaryCommand `yaml:"command,omitempty"`
	// Entrypoints overrides the default entry-point paths.
	// In split mode: ["cmd/api", "cmd/rpc"]. In single mode: ["cmd/service"].
	Entrypoints []string `yaml:"entrypoints,omitempty"`
}

// APITransportOptions are generation options for the API (Hertz) transport.
type APITransportOptions struct {
	// Port is the HTTP listen port written into etc/api.yaml.
	Port int `yaml:"port,omitempty"`
}

// RPCTransportOptions are generation options for the RPC (Kitex) transport.
type RPCTransportOptions struct {
	WithTrace bool `yaml:"with_trace,omitempty"`
	WithRedis bool `yaml:"with_redis,omitempty"`
}

// APITransportConfig describes the API (Hertz) transport in the manifest.
type APITransportConfig struct {
	Enabled bool                `yaml:"enabled"`
	// APIs lists .api definition files relative to the service directory.
	APIs    []string            `yaml:"apis,omitempty"`
	Options APITransportOptions `yaml:"options,omitempty"`
}

// RPCTransportConfig describes the RPC (Kitex) transport in the manifest.
type RPCTransportConfig struct {
	Enabled bool                `yaml:"enabled"`
	// Protos lists .proto definition files relative to the service directory.
	Protos  []string            `yaml:"protos,omitempty"`
	Options RPCTransportOptions `yaml:"options,omitempty"`
}

// TransportsConfig groups API and RPC transport configurations.
type TransportsConfig struct {
	API APITransportConfig `yaml:"api,omitempty"`
	RPC RPCTransportConfig `yaml:"rpc,omitempty"`
}

// ServiceManifest is the root of the service: section in .ss.yaml.
type ServiceManifest struct {
	// Binary controls compilation strategy.
	Binary BinaryConfig `yaml:"binary,omitempty"`
	// Transports declares which transports are active.
	Transports TransportsConfig `yaml:"transports,omitempty"`
	// Addons lists optional code-gen addons (e.g. "sqlc", "redis").
	Addons []string `yaml:"addons,omitempty"`
}

// IsHybrid returns true when both API and RPC transports are enabled.
func (m *ServiceManifest) IsHybrid() bool {
	return m.Transports.API.Enabled && m.Transports.RPC.Enabled
}

// ActiveTransports returns the list of enabled transport names.
func (m *ServiceManifest) ActiveTransports() []string {
	var transports []string
	if m.Transports.API.Enabled {
		transports = append(transports, "api")
	}
	if m.Transports.RPC.Enabled {
		transports = append(transports, "rpc")
	}
	return transports
}

// EffectiveBinaryMode returns the resolved binary mode (defaults to split).
func (m *ServiceManifest) EffectiveBinaryMode() BinaryMode {
	if m.Binary.Mode == BinaryModeSingle {
		return BinaryModeSingle
	}
	return BinaryModeSplit
}
