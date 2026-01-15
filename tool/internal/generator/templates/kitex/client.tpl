package {{.ServiceLower}}client

import (
	"context"
	"time"

	"github.com/cloudwego/kitex/client"
	"github.com/ssgohq/goten-core/logx"
	"github.com/ssgohq/goten-core/srpc"

	// TODO: Import your generated service client
	// "{{.Module}}/kitex_gen/{{.ServiceLower}}/{{.ServiceLower}}service"
)

// ClientConfig represents {{.Service}} client configuration.
type ClientConfig struct {
	srpc.ClientConfig `yaml:",inline"`
}

// Client wraps the {{.Service}} RPC client.
type Client struct {
	// TODO: Uncomment after kitex generates the service
	// cli {{.ServiceLower}}service.Client
}

// New creates a new {{.Service}} client.
func New(serviceName string, cfg *ClientConfig) (*Client, error) {
	cfg.ClientConfig.SetDefaults()

	opts := srpc.NewClientBuilder(&cfg.ClientConfig).Build()

	// TODO: Uncomment after kitex generates the service
	// cli, err := {{.ServiceLower}}service.NewClient(serviceName, opts...)
	// if err != nil {
	// 	return nil, err
	// }
	// return &Client{cli: cli}, nil

	_ = opts
	logx.Warnw("TODO: Implement client creation", "service", serviceName)
	return &Client{}, nil
}

// NewWithDiscovery creates a client using Consul service discovery.
func NewWithDiscovery(serviceName, consulAddr string) (*Client, error) {
	cfg := &ClientConfig{
		ClientConfig: srpc.ClientConfig{
			ServiceName: serviceName,
			Discovery: srpc.DiscoveryConfig{
				Type: "consul",
				Consul: srpc.ConsulConfig{
					Address: consulAddr,
				},
			},
		},
	}
	return New(serviceName, cfg)
}

// NewDirect creates a client with direct endpoint connection.
func NewDirect(serviceName string, endpoints []string) (*Client, error) {
	cfg := &ClientConfig{
		ClientConfig: srpc.ClientConfig{
			ServiceName: serviceName,
			Endpoints:   endpoints,
		},
	}
	return New(serviceName, cfg)
}

// WithTimeout returns a context with the specified timeout.
func WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}

// Example usage:
//
// func main() {
//     // Using service discovery
//     cli, err := {{.ServiceLower}}client.NewWithDiscovery("{{.ServiceLower}}-rpc", "localhost:8500")
//     if err != nil {
//         logx.Fatal("failed to create client", zap.Error(err))
//     }
//
//     // Make RPC call with timeout
//     ctx, cancel := {{.ServiceLower}}client.WithTimeout(context.Background(), 3*time.Second)
//     defer cancel()
//
//     resp, err := cli.GetUser(ctx, &{{.ServiceLower}}.GetUserRequest{Id: 123})
//     if err != nil {
//         logx.Errorw("RPC error", "error", err)
//         return
//     }
//     logx.Infow("User response", "user", resp.User)
// }