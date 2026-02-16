package schema

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newValidator(t *testing.T) *SchemaValidator {
	t.Helper()
	v, err := NewSchemaValidator()
	if err != nil {
		t.Fatalf("NewSchemaValidator failed: %v", err)
	}
	return v
}

func TestValidate_ReferenceExample(t *testing.T) {
	v := newValidator(t)

	examplePath := filepath.Join("..", "..", "schemas", "v1", "examples", "password-auth.allium.json")
	errors := v.Validate(examplePath)
	if len(errors) > 0 {
		t.Errorf("expected 0 errors for reference example, got %d:", len(errors))
		for _, e := range errors {
			t.Errorf("  %s", e)
		}
	}
}

func TestValidate_MissingVersion(t *testing.T) {
	v := newValidator(t)

	doc := map[string]any{
		"file": "test.allium",
	}
	errors := v.ValidateDocument(doc)
	if len(errors) == 0 {
		t.Fatal("expected errors for missing version")
	}

	found := false
	for _, e := range errors {
		if strings.Contains(e.Message, "version") || strings.Contains(e.Path, "version") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error mentioning 'version', got: %v", errors)
	}
}

func TestValidate_MissingFieldType(t *testing.T) {
	v := newValidator(t)

	// Entity field without "type" — violates Rule 2
	doc := map[string]any{
		"version": "1",
		"file":    "test.allium",
		"entities": []any{
			map[string]any{
				"name": "User",
				"fields": []any{
					map[string]any{
						"name": "email",
						// missing "type"
					},
				},
			},
		},
	}
	errors := v.ValidateDocument(doc)
	if len(errors) == 0 {
		t.Fatal("expected errors for field missing type (Rule 2)")
	}
}

func TestValidate_EmptyEnsures(t *testing.T) {
	v := newValidator(t)

	// Rule with empty ensures — violates Rule 4
	doc := map[string]any{
		"version": "1",
		"file":    "test.allium",
		"rules": []any{
			map[string]any{
				"name": "TestRule",
				"trigger": map[string]any{
					"kind":       "external_stimulus",
					"name":       "DoSomething",
					"parameters": []any{},
				},
				"ensures": []any{}, // empty — violates minItems: 1
			},
		},
	}
	errors := v.ValidateDocument(doc)
	if len(errors) == 0 {
		t.Fatal("expected errors for empty ensures (Rule 4)")
	}
}

func TestValidate_InvalidTriggerKind(t *testing.T) {
	v := newValidator(t)

	// Trigger with invalid kind — violates Rule 5
	doc := map[string]any{
		"version": "1",
		"file":    "test.allium",
		"rules": []any{
			map[string]any{
				"name": "TestRule",
				"trigger": map[string]any{
					"kind": "invalid_kind",
				},
				"ensures": []any{
					map[string]any{
						"kind": "state_change",
						"target": map[string]any{
							"kind":  "field_access",
							"field": "status",
						},
						"value": map[string]any{
							"kind":  "literal",
							"type":  "enum_value",
							"value": "active",
						},
					},
				},
			},
		},
	}
	errors := v.ValidateDocument(doc)
	if len(errors) == 0 {
		t.Fatal("expected errors for invalid trigger kind (Rule 5)")
	}
}

func TestValidate_SnakeCaseEntityName(t *testing.T) {
	v := newValidator(t)

	// Entity with snake_case name instead of PascalCase
	doc := map[string]any{
		"version": "1",
		"file":    "test.allium",
		"entities": []any{
			map[string]any{
				"name": "my_entity", // should be PascalCase
				"fields": []any{
					map[string]any{
						"name": "field_one",
						"type": map[string]any{
							"kind":  "primitive",
							"value": "String",
						},
					},
				},
			},
		},
	}
	errors := v.ValidateDocument(doc)
	if len(errors) == 0 {
		t.Fatal("expected errors for snake_case entity name")
	}
}

func TestValidate_ConfigMissingDefaultValue(t *testing.T) {
	v := newValidator(t)

	// Config param without default_value — violates Rule 25
	doc := map[string]any{
		"version": "1",
		"file":    "test.allium",
		"config": []any{
			map[string]any{
				"name": "max_retries",
				"type": map[string]any{
					"kind":  "primitive",
					"value": "Integer",
				},
				// missing default_value
			},
		},
	}
	errors := v.ValidateDocument(doc)
	if len(errors) == 0 {
		t.Fatal("expected errors for config missing default_value (Rule 25)")
	}
}

func TestValidate_ValidMinimalSpec(t *testing.T) {
	v := newValidator(t)

	doc := map[string]any{
		"version": "1",
		"file":    "test.allium",
	}
	errors := v.ValidateDocument(doc)
	if len(errors) > 0 {
		t.Errorf("expected 0 errors for minimal valid spec, got %d:", len(errors))
		for _, e := range errors {
			t.Errorf("  %s", e)
		}
	}
}

func TestValidate_NonexistentFile(t *testing.T) {
	v := newValidator(t)
	errors := v.Validate("/nonexistent/file.json")
	if len(errors) == 0 {
		t.Fatal("expected errors for nonexistent file")
	}
}

func TestValidate_InvalidJSON(t *testing.T) {
	v := newValidator(t)
	tmpFile := filepath.Join(t.TempDir(), "invalid.json")
	if err := os.WriteFile(tmpFile, []byte("{bad json}"), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	errors := v.Validate(tmpFile)
	if len(errors) == 0 {
		t.Fatal("expected errors for invalid JSON")
	}
}

func TestValidate_AdditionalProperties(t *testing.T) {
	v := newValidator(t)

	// Root document with unknown property
	doc := map[string]any{
		"version":          "1",
		"file":             "test.allium",
		"unknown_property": "should be rejected",
	}
	errors := v.ValidateDocument(doc)
	if len(errors) == 0 {
		t.Fatal("expected errors for additional properties")
	}
}

func TestValidate_VariantPascalCase(t *testing.T) {
	v := newValidator(t)

	// Variant with non-PascalCase name — violates Rule 15
	doc := map[string]any{
		"version": "1",
		"file":    "test.allium",
		"variants": []any{
			map[string]any{
				"name":        "my_variant", // should be PascalCase
				"base_entity": "Node",
				"fields":      []any{},
			},
		},
	}
	errors := v.ValidateDocument(doc)
	if len(errors) == 0 {
		t.Fatal("expected errors for non-PascalCase variant name (Rule 15)")
	}
}

func TestSchemaError_JSON(t *testing.T) {
	se := SchemaError{Path: "/entities/0/name", Message: "pattern mismatch"}
	data, err := json.Marshal(se)
	if err != nil {
		t.Fatalf("failed to marshal SchemaError: %v", err)
	}

	var decoded SchemaError
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal SchemaError: %v", err)
	}
	if decoded.Path != se.Path || decoded.Message != se.Message {
		t.Errorf("round-trip failed: got %+v, want %+v", decoded, se)
	}
}
