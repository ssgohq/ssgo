// Code scaffolded by ssgo. Safe to edit.

package config

import (
	"github.com/ssgohq/goten-core/metric"
	"github.com/ssgohq/goten-core/srpc"
{{- if .WithRedis}}
	"github.com/ssgohq/goten-core/stores/redis"
{{- end}}
)

// RPCConfig holds Kitex RPC transport configuration.
// It embeds BaseConfig for shared fields (Name, Log, Trace, Metric).
type RPCConfig struct {
	BaseConfig    `yaml:",inline"`
	srpc.ServerConfig `yaml:",inline"`
	Metric            metric.Config `yaml:"metric,omitempty"`
{{- if .WithRedis}}
	Redis redis.Config `yaml:"redis,omitempty"`
{{end}}
	// Database config will be added by 'ssgo dms bun gen'
	// Add your config here:
	// DB       BunConfig       `yaml:"db,omitempty"`
	// Redis    redis.Config    `yaml:"redis,omitempty"`
}

func (c RPCConfig) IsTraceEnabled() bool {
	return c.Trace.IsEnabled()
}

// Config is an alias for RPCConfig for backward compatibility.
type Config = RPCConfig
