// Code scaffolded by ssgo. Safe to edit.

package config

import (
	"github.com/ssgohq/goten-core/logx"
	"github.com/ssgohq/goten-core/metric"
	"github.com/ssgohq/goten-core/trace"
)

// BaseConfig holds shared configuration for all transports.
// It is embedded by transport-specific config structs.
type BaseConfig struct {
	Name   string        `yaml:"name"`
	Log    logx.Config   `yaml:"log,omitempty"`
	Trace  trace.Config  `yaml:"trace,omitempty"`
	Metric metric.Config `yaml:"metric,omitempty"`
}

// IsTraceEnabled returns true if tracing is enabled.
func (c BaseConfig) IsTraceEnabled() bool {
	return c.Trace.IsEnabled()
}
