// Package composer generates the single-binary entrypoint (cmd/server/main.go)
// that hosts both API and RPC transports in the same process.
package composer

// Options configures ComposerGenerator.
type Options struct {
	// OutputDir is the root directory of the service.
	OutputDir string

	// Module is the Go module name (e.g., "github.com/org/my-service").
	Module string

	// ServiceName is the human-readable service name (e.g., "MyService").
	ServiceName string

	// WithAPI enables API (Hertz) transport in the composed binary.
	WithAPI bool

	// WithRPC enables RPC (Kitex) transport in the composed binary.
	WithRPC bool

	// APIPort is the listen port for the API transport.
	APIPort int

	// RPCPort is the listen port for the RPC transport.
	RPCPort int

	// Verbose enables verbose logging during generation.
	Verbose bool
}

// Validate returns an error if required options are missing.
func (o Options) Validate() error {
	if o.OutputDir == "" {
		return errorf("OutputDir is required")
	}
	if o.Module == "" {
		return errorf("Module is required")
	}
	if !o.WithAPI && !o.WithRPC {
		return errorf("at least one of WithAPI or WithRPC must be enabled")
	}
	return nil
}

// templateData is the data passed to server_main.tpl.
type templateData struct {
	Module      string
	ServiceName string
	WithAPI     bool
	WithRPC     bool
	APIPort     int
	RPCPort     int
}

func buildTemplateData(o Options) templateData {
	apiPort := o.APIPort
	if apiPort == 0 {
		apiPort = 8080
	}
	rpcPort := o.RPCPort
	if rpcPort == 0 {
		rpcPort = 8888
	}
	return templateData{
		Module:      o.Module,
		ServiceName: o.ServiceName,
		WithAPI:     o.WithAPI,
		WithRPC:     o.WithRPC,
		APIPort:     apiPort,
		RPCPort:     rpcPort,
	}
}
