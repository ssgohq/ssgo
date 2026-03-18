package cmd

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ApiConfig represents the api: section of .ss.yaml.
//
// Example .ss.yaml:
//
//	api:
//	  apis:
//	    - file: api/user.api
//	      dir: .
//	      options:
//	        port: 8080
//	        with_logic: true
//	        format: json
type ApiConfig struct {
	Apis []ApiServiceConfig `yaml:"apis"`
}

// ApiServiceConfig represents a single API service entry in .ss.yaml.
type ApiServiceConfig struct {
	// File is the path to the .api file.
	File string `yaml:"file"`
	// Dir is the output directory (default: ".").
	Dir string `yaml:"dir"`
	// Options are generation options for this service.
	Options ApiServiceOptions `yaml:"options"`
}

// ApiServiceOptions are generation flags for an API service.
type ApiServiceOptions struct {
	// Port is the server port (default: 8080).
	Port int `yaml:"port"`
	// WithLogic controls whether logic files are generated (default: true).
	WithLogic *bool `yaml:"with_logic"`
	// Format is the doc output format: json|yaml (default: json).
	Format string `yaml:"format"`
}

// ssYamlApi is the full .ss.yaml structure (only api section is used here).
type ssYamlApi struct {
	Api ApiConfig `yaml:"api"`
}

// LoadApiConfig reads .ss.yaml from the given directory and returns its api section.
// Returns a zero-value ApiConfig (not an error) when .ss.yaml does not exist or
// does not have an api: section.
func LoadApiConfig(workDir string) (*ApiConfig, error) {
	configPath := filepath.Join(workDir, ".ss.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &ApiConfig{}, nil // Optional file — no error
		}
		return nil, err
	}

	var raw ssYamlApi
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	return &raw.Api, nil
}

// IsEmpty reports whether the ApiConfig has no meaningful configuration.
func (c *ApiConfig) IsEmpty() bool {
	return len(c.Apis) == 0
}
