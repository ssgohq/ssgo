package scanner

import "os"

// ServiceState describes what has already been generated in a service directory.
type ServiceState string

const (
	// ServiceStateEmpty means no recognisable generated layout was found.
	ServiceStateEmpty ServiceState = "empty"
	// ServiceStateAPIOnly means an API (Hertz) layout exists but no RPC.
	ServiceStateAPIOnly ServiceState = "api-only"
	// ServiceStateRPCOnly means an RPC (Kitex) layout exists but no API.
	ServiceStateRPCOnly ServiceState = "rpc-only"
	// ServiceStateHybrid means both API and RPC layouts exist.
	ServiceStateHybrid ServiceState = "hybrid"
	// ServiceStateHybridCapable means contracts for both transports exist
	// but neither has been generated yet.
	ServiceStateHybridCapable ServiceState = "hybrid-capable"
)

// Conflict describes a detected conflict or concern.
type Conflict struct {
	// Kind is a short machine-readable label.
	Kind string
	// Message is a human-readable description.
	Message string
}

// detectState examines a directory's layout (not contracts) to determine
// which transports have already been generated.
func detectState(dir string) ServiceState {
	hasAPI := dirExists(dir, "internal/api") || dirExists(dir, "internal/handler")
	hasRPC := dirExists(dir, "internal/rpc") || dirExists(dir, "internal/server") || dirExists(dir, "kitex_gen")

	switch {
	case hasAPI && hasRPC:
		return ServiceStateHybrid
	case hasAPI:
		return ServiceStateAPIOnly
	case hasRPC:
		return ServiceStateRPCOnly
	default:
		return ServiceStateEmpty
	}
}

// detectConflicts returns a list of conflicts and reuse opportunities for the
// given state and contract set.
func detectConflicts(state ServiceState, contracts []Contract) []Conflict {
	var conflicts []Conflict

	apiContracts := filterContracts(contracts, ContractKindAPI)
	rpcContracts := filterContracts(contracts, ContractKindProto)

	switch state {
	case ServiceStateAPIOnly:
		if len(rpcContracts) > 0 {
			conflicts = append(conflicts, Conflict{
				Kind:    "api-missing-rpc-layout",
				Message: "proto files found but no RPC layout generated; run gen --rpc to add RPC transport",
			})
		}
	case ServiceStateRPCOnly:
		if len(apiContracts) > 0 {
			conflicts = append(conflicts, Conflict{
				Kind:    "rpc-missing-api-layout",
				Message: ".api files found but no API layout generated; run gen --api to add API transport",
			})
		}
	case ServiceStateEmpty:
		if len(apiContracts) > 0 && len(rpcContracts) > 0 {
			conflicts = append(conflicts, Conflict{
				Kind:    "hybrid-capable-ungenerated",
				Message: "both .api and .proto contracts found but nothing has been generated yet",
			})
		}
	}

	return conflicts
}

// suggestGenerationOrder returns the transports in the recommended generation order.
func suggestGenerationOrder(contracts []Contract) []string {
	var order []string
	seen := map[string]bool{}

	// RPC first so kitex_gen is available when API code is compiled.
	if len(filterContracts(contracts, ContractKindProto)) > 0 {
		order = append(order, "rpc")
		seen["rpc"] = true
	}
	if len(filterContracts(contracts, ContractKindAPI)) > 0 {
		order = append(order, "api")
		seen["api"] = true
	}
	if len(filterContracts(contracts, ContractKindSQL)) > 0 {
		order = append(order, "sqlc")
	}
	return order
}

// filterContracts returns all contracts of the given kind.
func filterContracts(contracts []Contract, kind ContractKind) []Contract {
	var out []Contract
	for _, c := range contracts {
		if c.Kind == kind {
			out = append(out, c)
		}
	}
	return out
}

func dirExists(base, rel string) bool {
	info, err := os.Stat(joinPath(base, rel))
	return err == nil && info.IsDir()
}

func joinPath(base, rel string) string {
	if base == "" {
		return rel
	}
	return base + "/" + rel
}
