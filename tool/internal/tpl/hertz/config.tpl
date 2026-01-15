package config

import (
	"fmt"

	"github.com/ssgohq/goten-core/srpc"
	"github.com/ssgohq/goten-core/trace"
)

type Config struct {
	Name  string `yaml:"name"`
	Host  string `yaml:"host,omitempty"`
	Port  int    `yaml:"port,omitempty"`
	Trace trace.Config `yaml:"trace,omitempty"`

	{{if .HasRPC}}
	RPC map[string]srpc.ClientConfig `yaml:"rpc,omitempty"`
	{{end}}

	// Add your config here:
	// Redis    redis.Config    `yaml:"redis,omitempty"`
	// Postgres postgres.Config `yaml:"postgres,omitempty"`
}

// Addr returns the server address
func (c Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// IsTraceEnabled returns true if tracing is enabled
func (c Config) IsTraceEnabled() bool {
	return c.Trace.IsEnabled()
}