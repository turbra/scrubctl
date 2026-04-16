package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyConfigFile_ConfigOnly(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	writeFile(t, cfgPath, `includeKinds:
  - Deployment
  - Service
excludeKinds:
  - Secret
`)
	opts := &rootOptions{ConfigPath: cfgPath}
	cmd := newRootCommand()
	cmd.SetArgs([]string{"--config", cfgPath, "version"})
	// Parse flags so Changed() works correctly
	if err := cmd.ParseFlags([]string{"--config", cfgPath}); err != nil {
		t.Fatal(err)
	}
	if err := applyConfigFile(cmd, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.IncludeKinds != "Deployment,Service" {
		t.Errorf("IncludeKinds = %q, want %q", opts.IncludeKinds, "Deployment,Service")
	}
	if opts.ExcludeKinds != "Secret" {
		t.Errorf("ExcludeKinds = %q, want %q", opts.ExcludeKinds, "Secret")
	}
}

func TestApplyConfigFile_CLIOverridesConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	writeFile(t, cfgPath, `includeKinds:
  - Deployment
  - Service
excludeKinds:
  - Secret
`)
	opts := &rootOptions{ConfigPath: cfgPath, IncludeKinds: "ConfigMap", ExcludeKinds: "Route"}
	cmd := newRootCommand()
	// Simulate CLI flags being explicitly set
	if err := cmd.ParseFlags([]string{"--config", cfgPath, "--include-kinds", "ConfigMap", "--exclude-kinds", "Route"}); err != nil {
		t.Fatal(err)
	}
	if err := applyConfigFile(cmd, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.IncludeKinds != "ConfigMap" {
		t.Errorf("IncludeKinds = %q, want %q (CLI should override config)", opts.IncludeKinds, "ConfigMap")
	}
	if opts.ExcludeKinds != "Route" {
		t.Errorf("ExcludeKinds = %q, want %q (CLI should override config)", opts.ExcludeKinds, "Route")
	}
}

func TestApplyConfigFile_CLIOnlyNoConfig(t *testing.T) {
	opts := &rootOptions{IncludeKinds: "Deployment", ExcludeKinds: "Secret"}
	cmd := newRootCommand()
	if err := cmd.ParseFlags([]string{"--include-kinds", "Deployment", "--exclude-kinds", "Secret"}); err != nil {
		t.Fatal(err)
	}
	if err := applyConfigFile(cmd, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.IncludeKinds != "Deployment" {
		t.Errorf("IncludeKinds = %q, want %q", opts.IncludeKinds, "Deployment")
	}
	if opts.ExcludeKinds != "Secret" {
		t.Errorf("ExcludeKinds = %q, want %q", opts.ExcludeKinds, "Secret")
	}
}

func TestApplyConfigFile_MissingConfigFile(t *testing.T) {
	opts := &rootOptions{ConfigPath: "/nonexistent/config.yaml"}
	cmd := newRootCommand()
	if err := cmd.ParseFlags([]string{"--config", "/nonexistent/config.yaml"}); err != nil {
		t.Fatal(err)
	}
	err := applyConfigFile(cmd, opts)
	if err == nil {
		t.Fatal("expected error for missing config file")
	}
}

func TestApplyConfigFile_MalformedConfigFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "bad.yaml")
	writeFile(t, cfgPath, "includeKinds: not-a-list\n")
	opts := &rootOptions{ConfigPath: cfgPath}
	cmd := newRootCommand()
	if err := cmd.ParseFlags([]string{"--config", cfgPath}); err != nil {
		t.Fatal(err)
	}
	err := applyConfigFile(cmd, opts)
	if err == nil {
		t.Fatal("expected error for malformed config")
	}
}

func TestApplyConfigFile_EmptyConfigValues(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "empty.yaml")
	writeFile(t, cfgPath, "---\n")
	opts := &rootOptions{ConfigPath: cfgPath}
	cmd := newRootCommand()
	if err := cmd.ParseFlags([]string{"--config", cfgPath}); err != nil {
		t.Fatal(err)
	}
	if err := applyConfigFile(cmd, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.IncludeKinds != "" {
		t.Errorf("IncludeKinds = %q, want empty", opts.IncludeKinds)
	}
	if opts.ExcludeKinds != "" {
		t.Errorf("ExcludeKinds = %q, want empty", opts.ExcludeKinds)
	}
}

func TestApplyConfigFile_PartialOverride(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	writeFile(t, cfgPath, `includeKinds:
  - Deployment
  - Service
excludeKinds:
  - Secret
  - Route
`)
	opts := &rootOptions{ConfigPath: cfgPath, ExcludeKinds: "ConfigMap"}
	cmd := newRootCommand()
	// Only exclude-kinds set via CLI, include-kinds should come from config
	if err := cmd.ParseFlags([]string{"--config", cfgPath, "--exclude-kinds", "ConfigMap"}); err != nil {
		t.Fatal(err)
	}
	if err := applyConfigFile(cmd, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.IncludeKinds != "Deployment,Service" {
		t.Errorf("IncludeKinds = %q, want %q (should come from config)", opts.IncludeKinds, "Deployment,Service")
	}
	if opts.ExcludeKinds != "ConfigMap" {
		t.Errorf("ExcludeKinds = %q, want %q (CLI should override config)", opts.ExcludeKinds, "ConfigMap")
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
