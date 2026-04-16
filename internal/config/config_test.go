package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		wantInclude  []string
		wantExclude  []string
		wantErr      bool
		errSubstring string
	}{
		{
			name: "both fields",
			content: `includeKinds:
  - Deployment
  - Service
excludeKinds:
  - Secret
`,
			wantInclude: []string{"Deployment", "Service"},
			wantExclude: []string{"Secret"},
		},
		{
			name: "include only",
			content: `includeKinds:
  - ConfigMap
`,
			wantInclude: []string{"ConfigMap"},
			wantExclude: nil,
		},
		{
			name: "exclude only",
			content: `excludeKinds:
  - Route
  - BuildConfig
`,
			wantInclude: nil,
			wantExclude: []string{"Route", "BuildConfig"},
		},
		{
			name:        "empty file",
			content:     "",
			wantInclude: nil,
			wantExclude: nil,
		},
		{
			name:        "empty document",
			content:     "---\n",
			wantInclude: nil,
			wantExclude: nil,
		},
		{
			name:         "unknown field",
			content:      "foo: bar\n",
			wantErr:      true,
			errSubstring: "parsing config file",
		},
		{
			name:         "malformed yaml",
			content:      "includeKinds:\n  - [broken",
			wantErr:      true,
			errSubstring: "parsing config file",
		},
		{
			name: "wrong type for includeKinds",
			content: `includeKinds: "Deployment,Service"
`,
			wantErr:      true,
			errSubstring: "parsing config file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "config.yaml")
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}
			cfg, err := Load(path)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errSubstring != "" && !contains(err.Error(), tt.errSubstring) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errSubstring)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !sliceEqual(cfg.IncludeKinds, tt.wantInclude) {
				t.Errorf("IncludeKinds = %v, want %v", cfg.IncludeKinds, tt.wantInclude)
			}
			if !sliceEqual(cfg.ExcludeKinds, tt.wantExclude) {
				t.Errorf("ExcludeKinds = %v, want %v", cfg.ExcludeKinds, tt.wantExclude)
			}
		})
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !contains(err.Error(), "reading config file") {
		t.Errorf("error %q should contain 'reading config file'", err.Error())
	}
}

func TestCSVMethods(t *testing.T) {
	cfg := &File{
		IncludeKinds: []string{"Deployment", "Service", "ConfigMap"},
		ExcludeKinds: []string{"Secret"},
	}
	if got := cfg.IncludeKindsCSV(); got != "Deployment,Service,ConfigMap" {
		t.Errorf("IncludeKindsCSV() = %q, want %q", got, "Deployment,Service,ConfigMap")
	}
	if got := cfg.ExcludeKindsCSV(); got != "Secret" {
		t.Errorf("ExcludeKindsCSV() = %q, want %q", got, "Secret")
	}

	empty := &File{}
	if got := empty.IncludeKindsCSV(); got != "" {
		t.Errorf("empty IncludeKindsCSV() = %q, want empty", got)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func sliceEqual(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
