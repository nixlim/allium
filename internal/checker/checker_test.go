package checker

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/foundry-zero/allium/internal/ast"
	"github.com/foundry-zero/allium/internal/report"
	"github.com/foundry-zero/allium/internal/semantic"
)

var refExample = filepath.Join("..", "..", "schemas", "v1", "examples", "password-auth.allium.json")

func TestCheckReferenceExample(t *testing.T) {
	c, err := NewChecker()
	if err != nil {
		t.Fatalf("NewChecker: %v", err)
	}

	r := c.Check(refExample, CheckOptions{})

	if !r.SchemaValid {
		t.Error("expected SchemaValid=true for reference example")
	}

	// The reference example should pass all validations cleanly.
	for _, e := range r.Errors {
		t.Errorf("unexpected error: [%s] %s at %s", e.Rule, e.Message, e.Location.Path)
	}
	for _, w := range r.Warnings {
		t.Errorf("unexpected warning: [%s] %s at %s", w.Rule, w.Message, w.Location.Path)
	}
}

func TestCheckReferenceExampleSemanticOnly(t *testing.T) {
	c, err := NewChecker()
	if err != nil {
		t.Fatalf("NewChecker: %v", err)
	}

	// Run only core semantic passes (references, uniqueness, expressions, sumtypes)
	// These should produce zero errors on the reference example.
	r := c.Check(refExample, CheckOptions{RuleFilter: []int{1, 3, 6, 10, 11, 12, 13, 14, 16, 17, 18, 19, 22, 23, 26, 27, 28, 30, 31, 35}})

	if !r.SchemaValid {
		t.Error("expected SchemaValid=true")
	}
	if r.HasErrors() {
		t.Errorf("expected 0 errors from core passes, got %d:", r.Summary.ErrorCount)
		for _, e := range r.Errors {
			t.Logf("  [%s] %s at %s", e.Rule, e.Message, e.Location.Path)
		}
	}
}

func TestCheckSchemaOnly(t *testing.T) {
	c, err := NewChecker()
	if err != nil {
		t.Fatalf("NewChecker: %v", err)
	}

	r := c.Check(refExample, CheckOptions{SchemaOnly: true})

	if !r.SchemaValid {
		t.Error("expected SchemaValid=true")
	}
	// SchemaOnly should still produce 0 errors on a valid file
	if r.HasErrors() {
		t.Errorf("expected 0 errors with SchemaOnly, got %d", r.Summary.ErrorCount)
	}
}

func TestCheckRuleFilter(t *testing.T) {
	c, err := NewChecker()
	if err != nil {
		t.Fatalf("NewChecker: %v", err)
	}

	// Filter for state machine rules only (7,8,9)
	// No state machine pass is registered yet, so no semantic errors should appear.
	r := c.Check(refExample, CheckOptions{RuleFilter: []int{7, 8, 9}})

	if !r.SchemaValid {
		t.Error("expected SchemaValid=true")
	}
	// The reference example is valid, so no errors expected
	if r.HasErrors() {
		t.Errorf("expected 0 errors with RuleFilter, got %d", r.Summary.ErrorCount)
	}
}

func TestCheckNonexistentFile(t *testing.T) {
	c, err := NewChecker()
	if err != nil {
		t.Fatalf("NewChecker: %v", err)
	}

	r := c.Check("/nonexistent/path/to/file.json", CheckOptions{})

	if r.File != "/nonexistent/path/to/file.json" {
		t.Errorf("expected file path in report, got %q", r.File)
	}
	if !r.HasErrors() {
		t.Error("expected an error for nonexistent file")
	}
	// Verify at least one INPUT error
	foundInput := false
	for _, e := range r.Errors {
		if e.Rule == "INPUT" {
			foundInput = true
			break
		}
	}
	if !foundInput {
		t.Error("expected an INPUT error for nonexistent file")
	}
}

func TestCheckSchemaErrors(t *testing.T) {
	c, err := NewChecker()
	if err != nil {
		t.Fatalf("NewChecker: %v", err)
	}

	// Create a temp file with valid JSON but invalid schema (missing required "version")
	dir := t.TempDir()
	path := filepath.Join(dir, "bad-schema.allium.json")
	if err := os.WriteFile(path, []byte(`{"file": "test.allium"}`), 0644); err != nil {
		t.Fatal(err)
	}

	r := c.Check(path, CheckOptions{})

	if r.SchemaValid {
		t.Error("expected SchemaValid=false for invalid schema")
	}
	if !r.HasErrors() {
		t.Error("expected schema errors")
	}
}

func TestCheckSchemaErrorsSkipSemantic(t *testing.T) {
	c, err := NewChecker()
	if err != nil {
		t.Fatalf("NewChecker: %v", err)
	}

	// Add a sentinel pass that records whether it was called.
	semanticCalled := false
	c.RegisterPass("sentinel", []int{1}, func(_ *ast.Spec, _ *semantic.SymbolTable) []report.Finding {
		semanticCalled = true
		return nil
	})

	dir := t.TempDir()
	path := filepath.Join(dir, "bad-schema.allium.json")
	if err := os.WriteFile(path, []byte(`{"file": "test.allium"}`), 0644); err != nil {
		t.Fatal(err)
	}

	r := c.Check(path, CheckOptions{})

	if r.SchemaValid {
		t.Error("expected SchemaValid=false")
	}
	if semanticCalled {
		t.Error("semantic pass should NOT run when schema validation fails")
	}
	// All errors should be SCHEMA errors
	for _, e := range r.Errors {
		if e.Rule != "SCHEMA" {
			t.Errorf("expected only SCHEMA errors when schema is invalid, got rule %q", e.Rule)
		}
	}
}

func TestPassMatchesFilter(t *testing.T) {
	tests := []struct {
		name      string
		passRules []int
		filter    []int
		want      bool
	}{
		{"empty filter runs all", []int{1, 3}, nil, true},
		{"matching rule", []int{7, 8, 9}, []int{8}, true},
		{"no matching rule", []int{7, 8, 9}, []int{1, 3}, false},
		{"partial overlap", []int{1, 3, 22}, []int{3, 7}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := passMatchesFilter(tt.passRules, tt.filter)
			if got != tt.want {
				t.Errorf("passMatchesFilter(%v, %v) = %v, want %v",
					tt.passRules, tt.filter, got, tt.want)
			}
		})
	}
}
