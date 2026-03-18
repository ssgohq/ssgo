package scanner

import "fmt"

// ScanResult is the output of scanning a directory.
type ScanResult struct {
	// Root is the absolute path that was scanned.
	Root string
	// Contracts is the full list of discovered contract files.
	Contracts []Contract
	// State is the current generated-layout state of the directory.
	State ServiceState
	// SuggestedOrder is the recommended generation sequence.
	SuggestedOrder []string
	// Conflicts lists detected conflicts and reuse opportunities.
	Conflicts []Conflict
}

// APIContracts returns only API (.api) contracts from the result.
func (r *ScanResult) APIContracts() []Contract {
	return filterContracts(r.Contracts, ContractKindAPI)
}

// RPCContracts returns only RPC (.proto) contracts from the result.
func (r *ScanResult) RPCContracts() []Contract {
	return filterContracts(r.Contracts, ContractKindProto)
}

// SQLContracts returns only SQL contracts from the result.
func (r *ScanResult) SQLContracts() []Contract {
	return filterContracts(r.Contracts, ContractKindSQL)
}

// HasConflicts returns true when at least one conflict was detected.
func (r *ScanResult) HasConflicts() bool {
	return len(r.Conflicts) > 0
}

// ScanDir scans root for contract files and existing generated layout,
// then returns a ScanResult describing what was found.
func ScanDir(root string) (*ScanResult, error) {
	if root == "" {
		return nil, fmt.Errorf("root directory must not be empty")
	}

	contracts, err := findContracts(root)
	if err != nil {
		return nil, fmt.Errorf("scan %s: %w", root, err)
	}

	state := detectState(root)

	// Refine empty state: if we have contracts for both transports,
	// the directory is hybrid-capable even though nothing is generated.
	if state == ServiceStateEmpty {
		hasAPI := len(filterContracts(contracts, ContractKindAPI)) > 0
		hasRPC := len(filterContracts(contracts, ContractKindProto)) > 0
		if hasAPI && hasRPC {
			state = ServiceStateHybridCapable
		}
	}

	conflicts := detectConflicts(state, contracts)
	order := suggestGenerationOrder(contracts)

	return &ScanResult{
		Root:           root,
		Contracts:      contracts,
		State:          state,
		SuggestedOrder: order,
		Conflicts:      conflicts,
	}, nil
}
