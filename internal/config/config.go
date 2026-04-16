package config

import (
	"fmt"
	"os"
	"strings"

	"sigs.k8s.io/yaml"
)

// File represents the structure of a scrubctl config file.
type File struct {
	IncludeKinds []string `json:"includeKinds,omitempty"`
	ExcludeKinds []string `json:"excludeKinds,omitempty"`
}

// Load reads and parses a config file from the given path.
func Load(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}
	var cfg File
	if err := yaml.UnmarshalStrict(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}
	return &cfg, nil
}

// IncludeKindsCSV returns the includeKinds list as a comma-separated string.
func (f *File) IncludeKindsCSV() string {
	return strings.Join(f.IncludeKinds, ",")
}

// ExcludeKindsCSV returns the excludeKinds list as a comma-separated string.
func (f *File) ExcludeKindsCSV() string {
	return strings.Join(f.ExcludeKinds, ",")
}
