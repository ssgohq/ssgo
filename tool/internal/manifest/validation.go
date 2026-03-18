package manifest

import (
	"fmt"
	"strings"
)

// ValidationError collects all manifest validation failures.
type ValidationError struct {
	Errors []string
}

func (e *ValidationError) Error() string {
	var b strings.Builder
	b.WriteString("invalid service manifest:")
	for _, err := range e.Errors {
		b.WriteString("\n  - ")
		b.WriteString(err)
	}
	return b.String()
}

func (e *ValidationError) add(format string, args ...any) {
	e.Errors = append(e.Errors, fmt.Sprintf(format, args...))
}

func (e *ValidationError) hasErrors() bool {
	return len(e.Errors) > 0
}

// Validate checks the manifest for invalid or contradictory states.
// It returns nil when the manifest is valid, or a *ValidationError describing
// all detected problems.
func Validate(m *ServiceManifest) error {
	ve := &ValidationError{}

	validateBinary(m, ve)
	validateTransports(m, ve)
	validateAddons(m, ve)

	if ve.hasErrors() {
		return ve
	}
	return nil
}

func validateBinary(m *ServiceManifest, ve *ValidationError) {
	switch m.Binary.Mode {
	case "", BinaryModeSplit, BinaryModeSingle:
		// valid
	default:
		ve.add("binary.mode %q is not valid; must be %q or %q",
			m.Binary.Mode, BinaryModeSplit, BinaryModeSingle)
	}

	if m.Binary.Mode == BinaryModeSingle {
		switch m.Binary.Command {
		case "", BinaryCommandAPI, BinaryCommandRPC, BinaryCommandService, BinaryCommandBoth:
			// valid
		default:
			ve.add("binary.command %q is not valid for single mode; must be one of: api, rpc, service, both",
				m.Binary.Command)
		}

		// single mode requires at least one transport to be enabled so it knows what to run.
		if !m.Transports.API.Enabled && !m.Transports.RPC.Enabled {
			ve.add("binary.mode=single requires at least one transport (transports.api or transports.rpc) to be enabled")
		}
	}
}

func validateTransports(m *ServiceManifest, ve *ValidationError) {
	api := m.Transports.API
	rpc := m.Transports.RPC

	if !api.Enabled && !rpc.Enabled {
		// Empty manifest is allowed (service section may be absent).
		return
	}

	if api.Enabled && len(api.APIs) == 0 {
		ve.add("transports.api.enabled=true but no .api files listed in transports.api.apis")
	}

	if rpc.Enabled && len(rpc.Protos) == 0 {
		ve.add("transports.rpc.enabled=true but no .proto files listed in transports.rpc.protos")
	}

	if api.Options.Port < 0 || api.Options.Port > 65535 {
		ve.add("transports.api.options.port %d is out of valid range [0, 65535]", api.Options.Port)
	}
}

func validateAddons(m *ServiceManifest, ve *ValidationError) {
	known := map[string]bool{
		"sqlc":  true,
		"redis": true,
		"bun":   true,
		"gorm":  true,
	}
	seen := map[string]bool{}
	for _, addon := range m.Addons {
		if !known[addon] {
			ve.add("addon %q is not recognized; known addons: sqlc, redis, bun, gorm", addon)
		}
		if seen[addon] {
			ve.add("addon %q is listed more than once", addon)
		}
		seen[addon] = true
	}
}
