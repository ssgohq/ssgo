// Package runner provides process and service management for ssgo run command.
package runner

import (
	"errors"
	"sort"
	"time"
)

// Common errors
var (
	ErrNoServices = errors.New("no services configured")
)

// Options holds configuration for creating a new Runner.
type Options struct {
	Config   *RunnerConfig
	WorkDir  string
	Services []string
	NoWatch  bool
	NoBuild  bool
	NoTUI    bool
	Verbose  bool
}

// RunnerConfig holds the configuration for the run command.
type RunnerConfig struct {
	// Services to run
	Services []ServiceConfig

	// BuildDelay is the debounce delay for file changes
	BuildDelay time.Duration

	// KillDelay is the time to wait for graceful shutdown
	KillDelay time.Duration
}

// FilterServices returns services matching the given names.
// If names is empty, returns all services.
func (c *RunnerConfig) FilterServices(names []string) []ServiceConfig {
	if len(names) == 0 {
		return c.Services
	}

	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}

	var filtered []ServiceConfig
	for _, svc := range c.Services {
		if nameSet[svc.Name] {
			filtered = append(filtered, svc)
		}
	}
	return filtered
}

// TopologicalSort sorts services by dependencies.
// Services with no dependencies come first, then services that depend on them.
func (c *RunnerConfig) TopologicalSort(services []ServiceConfig) ([]ServiceConfig, error) {
	if len(services) == 0 {
		return nil, nil
	}

	// Build dependency graph
	serviceMap := make(map[string]ServiceConfig)
	for _, svc := range services {
		serviceMap[svc.Name] = svc
	}

	// Kahn's algorithm for topological sort
	inDegree := make(map[string]int)
	for _, svc := range services {
		if _, ok := inDegree[svc.Name]; !ok {
			inDegree[svc.Name] = 0
		}
		for _, dep := range svc.DependsOn {
			// Only count dependencies within our service set
			if _, ok := serviceMap[dep]; ok {
				inDegree[svc.Name]++
			}
		}
	}

	// Start with services that have no dependencies
	var queue []string
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}
	sort.Strings(queue) // Stable ordering

	var result []ServiceConfig
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]

		if svc, ok := serviceMap[name]; ok {
			result = append(result, svc)
		}

		// Reduce in-degree for dependents
		for _, svc := range services {
			for _, dep := range svc.DependsOn {
				if dep == name {
					inDegree[svc.Name]--
					if inDegree[svc.Name] == 0 {
						queue = append(queue, svc.Name)
					}
				}
			}
		}
		sort.Strings(queue) // Maintain stable ordering
	}

	if len(result) != len(services) {
		return nil, errors.New("circular dependency detected")
	}

	return result, nil
}

// ServiceConfig holds configuration for a single service.
type ServiceConfig struct {
	// Name of the service
	Name string

	// Working directory (relative to project root)
	Dir string

	// Build command (optional)
	Build string

	// Run command
	Cmd string
	Run string // Alternative field name

	// Color for logging
	Color string

	// Environment variables
	Env []string

	// Dependencies (other service names)
	DependsOn []string

	// Watch configuration
	Watch WatchConfig
}

// WatchConfig holds file watching configuration.
type WatchConfig struct {
	// Include patterns (glob)
	Include []string

	// Exclude patterns (glob)
	Exclude []string
}
