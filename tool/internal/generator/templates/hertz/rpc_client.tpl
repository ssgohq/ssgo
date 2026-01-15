package rpc

import (
	"fmt"
	"time"

	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/pkg/retry"
	kitextracing "github.com/kitex-contrib/obs-opentelemetry/tracing"
	"github.com/ssgohq/goten-core/logx"
	"github.com/ssgohq/goten-core/srpc"
	"{{.Module}}/internal/config"
	"{{.TypesModule}}/kitex_gen/{{.Package}}/{{.ServiceLower}}"
)

// {{.ServiceName}}Client wraps the Kitex client for {{.ServiceName}} RPC service
type {{.ServiceName}}Client struct {
	cli {{.ServiceLower}}.Client
}

// New{{.ServiceName}}Client creates a new {{.ServiceName}} RPC client with the given configuration.
// It supports both direct connection and Consul-based service discovery.
// Tracing is automatically enabled using the global TracerProvider.
func New{{.ServiceName}}Client(cfg config.{{.ServiceName}}RPCConfig) *{{.ServiceName}}Client {
	cfg.SetDefaults()

	var opts []client.Option

	// Add OpenTelemetry tracing suite for distributed tracing
	// This automatically propagates trace context to downstream services
	opts = append(opts, client.WithSuite(kitextracing.NewClientSuite()))

	// Use srpc helper for building client options
	if cfg.Discovery.Type == "consul" || cfg.Discovery.Type == "etcd" {
		// Use service discovery
		clientCfg := cfg.ToClientConfig()
		opts = append(opts, srpc.BuildClient(clientCfg)...)
		logx.Infow("Creating {{.ServiceName}}Client with service discovery",
			"serviceName", cfg.ServiceName,
			"discoveryType", cfg.Discovery.Type,
			"tracing", "enabled",
		)
	} else if len(cfg.Endpoints) > 0 {
		// Direct connection mode
		opts = append(opts,
			client.WithHostPorts(cfg.Endpoints...),
			client.WithRPCTimeout(cfg.Timeout.RPC),
			client.WithConnectTimeout(cfg.Timeout.Connect),
		)
		if cfg.Retry.Enabled {
			fp := retry.NewFailurePolicy()
			fp.WithMaxRetryTimes(cfg.Retry.MaxRetries)
			opts = append(opts, client.WithFailureRetry(fp))
		}
		logx.Infow("Creating {{.ServiceName}}Client with direct connection",
			"endpoints", cfg.Endpoints,
			"tracing", "enabled",
		)
	} else {
		panic("{{.ServiceName}}Client: either discovery or endpoints must be configured")
	}

	cli, err := {{.ServiceLower}}.NewClient(cfg.ServiceName, opts...)
	if err != nil {
		panic(fmt.Sprintf("failed to create {{.ServiceName}}Client: %v", err))
	}

	return &{{.ServiceName}}Client{cli: cli}
}

// New{{.ServiceName}}ClientDirect creates a {{.ServiceName}} RPC client with direct connection to the given address.
// This is useful for simple setups without service discovery.
func New{{.ServiceName}}ClientDirect(addr string) *{{.ServiceName}}Client {
	cli, err := {{.ServiceLower}}.NewClient(
		"{{.Package}}-rpc",
		client.WithHostPorts(addr),
		client.WithRPCTimeout(3*time.Second),
		client.WithConnectTimeout(1*time.Second),
		client.WithFailureRetry(retry.NewFailurePolicy()),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create {{.ServiceName}} client: %v", err))
	}
	return &{{.ServiceName}}Client{cli: cli}
}

// Client returns the underlying Kitex client
func (c *{{.ServiceName}}Client) Client() {{.ServiceLower}}.Client {
	return c.cli
}