// Package addon implements the plugin/marker-based addon pipeline for service
// generation. Addons extend the transport generation steps with optional
// code-gen targets such as sqlc, redis stubs, and tracing helpers.
package addon

import "fmt"

// AddonDef describes a single addon — its detection logic and execution.
type AddonDef struct {
	// Name is the unique addon identifier (e.g. "sqlc", "redis", "tracing").
	Name string
	// Detect reports whether the addon is applicable to the given directory.
	// It should return true when the relevant markers are present.
	Detect func(dir string) bool
	// Run executes the addon for the given directory and options.
	Run func(dir string, opts RunOpts) error
}

// RegisteredAddons is the ordered registry of known addons.
// Addons are executed in registration order for determinism.
var RegisteredAddons = []AddonDef{
	{
		Name:   "sqlc",
		Detect: detectSQLC,
		Run:    runSQLC,
	},
	{
		Name:   "redis",
		Detect: detectRedis,
		Run:    runRedis,
	},
	{
		Name:   "tracing",
		Detect: detectTracing,
		Run:    runTracing,
	},
}

// addonByName returns the AddonDef for the given name, or an error.
func addonByName(name string) (AddonDef, error) {
	for _, a := range RegisteredAddons {
		if a.Name == name {
			return a, nil
		}
	}
	return AddonDef{}, fmt.Errorf("addon %q is not registered", name)
}

// KnownAddonNames returns the list of all registered addon names.
func KnownAddonNames() []string {
	names := make([]string, 0, len(RegisteredAddons))
	for _, a := range RegisteredAddons {
		names = append(names, a.Name)
	}
	return names
}
