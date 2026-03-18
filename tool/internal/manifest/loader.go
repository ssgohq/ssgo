package manifest

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ssYamlWithService is the full .ss.yaml structure for service section loading.
type ssYamlWithService struct {
	Service ServiceManifest `yaml:"service"`
}

// Load reads the service: section from .ss.yaml in the given directory.
// Returns a zero-value ServiceManifest (not an error) when .ss.yaml does not exist
// or does not have a service: section.
func Load(workDir string) (*ServiceManifest, error) {
	configPath := filepath.Join(workDir, ".ss.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &ServiceManifest{}, nil
		}
		return nil, fmt.Errorf("read .ss.yaml: %w", err)
	}

	var raw ssYamlWithService
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse .ss.yaml: %w", err)
	}
	return &raw.Service, nil
}

// Save writes the service manifest back into .ss.yaml, merging with any
// existing content (other top-level keys are preserved).
func Save(workDir string, m *ServiceManifest) error {
	configPath := filepath.Join(workDir, ".ss.yaml")

	// Load existing raw content to preserve other sections.
	rawContent := make(map[string]any)
	if data, err := os.ReadFile(configPath); err == nil {
		_ = yaml.Unmarshal(data, &rawContent)
	}

	// Marshal the manifest to a generic map so yaml.v3 round-trips cleanly.
	manifestBytes, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal service manifest: %w", err)
	}
	var manifestMap any
	if err := yaml.Unmarshal(manifestBytes, &manifestMap); err != nil {
		return fmt.Errorf("unmarshal service manifest map: %w", err)
	}
	rawContent["service"] = manifestMap

	out, err := yaml.Marshal(rawContent)
	if err != nil {
		return fmt.Errorf("marshal .ss.yaml: %w", err)
	}
	return os.WriteFile(configPath, out, 0o644)
}
