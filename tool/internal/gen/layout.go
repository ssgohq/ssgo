package gen

import (
	"os"
	"path/filepath"
)

// ServiceLayout describes the transport layout of a generated service.
type ServiceLayout struct {
	HasAPI    bool   // cmd/api/ exists
	HasRPC    bool   // cmd/rpc/ exists
	IsLegacy  bool   // only cmd/main.go (no transport subdir)
	IsHybrid  bool   // both API and RPC present
	OutputDir string // service root directory
}

// DetectLayout inspects the filesystem to determine the current service layout.
func DetectLayout(outputDir string) ServiceLayout {
	layout := ServiceLayout{OutputDir: outputDir}

	hasAPICmdDir := dirExists(filepath.Join(outputDir, "cmd", "api"))
	hasRPCCmdDir := dirExists(filepath.Join(outputDir, "cmd", "rpc"))
	hasLegacyMain := fileExists(filepath.Join(outputDir, "cmd", "main.go"))

	layout.HasAPI = hasAPICmdDir
	layout.HasRPC = hasRPCCmdDir
	layout.IsHybrid = hasAPICmdDir && hasRPCCmdDir

	// Legacy: only cmd/main.go exists, no transport subdirs
	if hasLegacyMain && !hasAPICmdDir && !hasRPCCmdDir {
		layout.IsLegacy = true
	}

	return layout
}

// TransportDir returns the transport-specific subdirectory for internal code.
// For namespaced layout: "api" or "rpc"
// For legacy layout: "" (empty — files go directly in internal/)
func (l ServiceLayout) TransportDir(transport string) string {
	if l.IsLegacy {
		return ""
	}
	return transport
}

// CmdDir returns the cmd directory for a transport.
// Namespaced: "cmd/api" or "cmd/rpc"
// Legacy: "cmd"
func (l ServiceLayout) CmdDir(transport string) string {
	if l.IsLegacy {
		return "cmd"
	}
	return filepath.Join("cmd", transport)
}

// dirExists returns true if the path exists and is a directory.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// fileExists returns true if the path exists and is a regular file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
