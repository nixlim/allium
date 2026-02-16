package ast

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadSpec_ValidFile(t *testing.T) {
	// Find the reference example relative to the module root
	examplePath := filepath.Join("..", "..", "schemas", "v1", "examples", "password-auth.allium.json")

	spec, err := LoadSpec(examplePath)
	if err != nil {
		t.Fatalf("LoadSpec returned error: %v", err)
	}
	if spec == nil {
		t.Fatal("LoadSpec returned nil spec")
	}

	// Verify version
	if spec.Version != "1" {
		t.Errorf("expected version '1', got %q", spec.Version)
	}

	// Verify file
	if spec.File != "password-auth.allium" {
		t.Errorf("expected file 'password-auth.allium', got %q", spec.File)
	}

	// Verify entities are non-empty
	if len(spec.Entities) == 0 {
		t.Error("expected non-empty Entities")
	}

	// Verify rules are non-empty
	if len(spec.Rules) == 0 {
		t.Error("expected non-empty Rules")
	}

	// Verify specific entity names
	entityNames := make(map[string]bool)
	for _, e := range spec.Entities {
		entityNames[e.Name] = true
	}
	for _, name := range []string{"User", "Session", "PasswordResetToken"} {
		if !entityNames[name] {
			t.Errorf("expected entity %q not found", name)
		}
	}

	// Verify external entities
	if len(spec.ExternalEntities) == 0 {
		t.Error("expected non-empty ExternalEntities")
	}

	// Verify config
	if len(spec.Config) == 0 {
		t.Error("expected non-empty Config")
	}

	// Verify actors
	if len(spec.Actors) == 0 {
		t.Error("expected non-empty Actors")
	}

	// Verify surfaces
	if len(spec.Surfaces) == 0 {
		t.Error("expected non-empty Surfaces")
	}

	// Verify given bindings
	if len(spec.Given) == 0 {
		t.Error("expected non-empty Given")
	}

	// Verify defaults
	if len(spec.Defaults) == 0 {
		t.Error("expected non-empty Defaults")
	}

	// Verify enumerations
	if len(spec.Enumerations) == 0 {
		t.Error("expected non-empty Enumerations")
	}

	// Verify all 7 trigger types are present
	triggerKinds := make(map[string]bool)
	for _, r := range spec.Rules {
		triggerKinds[r.Trigger.Kind] = true
	}
	expectedTriggers := []string{
		"external_stimulus", "state_transition", "state_becomes",
		"temporal", "derived_condition", "entity_creation", "chained",
	}
	for _, tk := range expectedTriggers {
		if !triggerKinds[tk] {
			t.Errorf("expected trigger kind %q not found", tk)
		}
	}

	// Verify all 8 ensures clause types are present
	ensuresKinds := make(map[string]bool)
	var collectEnsuresKinds func(clauses []EnsuresClause)
	collectEnsuresKinds = func(clauses []EnsuresClause) {
		for _, c := range clauses {
			ensuresKinds[c.Kind] = true
			collectEnsuresKinds(c.Then)
			collectEnsuresKinds(c.Else)
			collectEnsuresKinds(c.Body)
		}
	}
	for _, r := range spec.Rules {
		collectEnsuresKinds(r.Ensures)
	}
	expectedEnsures := []string{
		"state_change", "entity_creation", "trigger_emission",
		"entity_removal", "conditional", "iteration", "let_binding", "set_mutation",
	}
	for _, ek := range expectedEnsures {
		if !ensuresKinds[ek] {
			t.Errorf("expected ensures kind %q not found", ek)
		}
	}
}

func TestLoadSpec_NonexistentFile(t *testing.T) {
	spec, err := LoadSpec("/nonexistent/path/to/file.json")
	if spec != nil {
		t.Error("expected nil spec for nonexistent file")
	}
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "no such file") && !strings.Contains(errMsg, "file not found") && !os.IsNotExist(err) {
		// On different OS, the error message varies
		if !strings.Contains(errMsg, "read spec file") {
			t.Errorf("expected error about file not found, got: %v", err)
		}
	}
}

func TestLoadSpec_InvalidJSON(t *testing.T) {
	// Write a temporary file with invalid JSON
	tmpFile := filepath.Join(t.TempDir(), "invalid.json")
	if err := os.WriteFile(tmpFile, []byte("{invalid json}"), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	spec, err := LoadSpec(tmpFile)
	if spec != nil {
		t.Error("expected nil spec for invalid JSON")
	}
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "parse") && !strings.Contains(errMsg, "unmarshal") {
		t.Errorf("expected error about parse/unmarshal, got: %v", err)
	}
}

func TestLoadSpec_EmptySpec(t *testing.T) {
	// Write a minimal valid JSON that's a valid spec
	tmpFile := filepath.Join(t.TempDir(), "minimal.json")
	content := `{"version": "1", "file": "test.allium"}`
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	spec, err := LoadSpec(tmpFile)
	if err != nil {
		t.Fatalf("LoadSpec returned error: %v", err)
	}
	if spec == nil {
		t.Fatal("LoadSpec returned nil spec")
	}
	if spec.Version != "1" {
		t.Errorf("expected version '1', got %q", spec.Version)
	}
	if spec.File != "test.allium" {
		t.Errorf("expected file 'test.allium', got %q", spec.File)
	}
}
