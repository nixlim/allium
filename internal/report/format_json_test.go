package report

import (
	"encoding/json"
	"testing"
)

func TestFormatJSONEmpty(t *testing.T) {
	r := NewReport("clean.allium.json")
	r.SchemaValid = true

	data, err := FormatJSON(r)
	if err != nil {
		t.Fatalf("FormatJSON: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if m["file"] != "clean.allium.json" {
		t.Errorf("file = %v", m["file"])
	}
	if m["schema_valid"] != true {
		t.Errorf("schema_valid = %v", m["schema_valid"])
	}

	// errors and warnings should be empty arrays, not null
	for _, key := range []string{"errors", "warnings"} {
		arr, ok := m[key].([]any)
		if !ok {
			t.Errorf("%q should be an array", key)
			continue
		}
		if len(arr) != 0 {
			t.Errorf("%q should be empty", key)
		}
	}

	summary, ok := m["summary"].(map[string]any)
	if !ok {
		t.Fatal("summary missing or wrong type")
	}
	if summary["error_count"] != float64(0) {
		t.Errorf("error_count = %v", summary["error_count"])
	}
	if summary["warning_count"] != float64(0) {
		t.Errorf("warning_count = %v", summary["warning_count"])
	}
}

func TestFormatJSONWithFindings(t *testing.T) {
	r := NewReport("bad.allium.json")
	r.SchemaValid = false
	r.AddFinding(NewError("RULE-01", "entity not found", Location{
		File: "bad.allium.json",
		Path: "$.entities[0]",
		Line: 10,
	}))
	r.AddFinding(NewWarning("WARN-01", "unused entity", Location{
		File: "bad.allium.json",
		Path: "$.entities[1]",
	}))

	data, err := FormatJSON(r)
	if err != nil {
		t.Fatalf("FormatJSON: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	errors, ok := m["errors"].([]any)
	if !ok || len(errors) != 1 {
		t.Fatalf("errors should have 1 item, got %v", m["errors"])
	}
	errObj := errors[0].(map[string]any)
	if errObj["rule"] != "RULE-01" {
		t.Errorf("error rule = %v", errObj["rule"])
	}
	if errObj["severity"] != "error" {
		t.Errorf("error severity = %v", errObj["severity"])
	}
	loc := errObj["location"].(map[string]any)
	if loc["line"] != float64(10) {
		t.Errorf("error line = %v", loc["line"])
	}

	warnings, ok := m["warnings"].([]any)
	if !ok || len(warnings) != 1 {
		t.Fatalf("warnings should have 1 item, got %v", m["warnings"])
	}
	warnObj := warnings[0].(map[string]any)
	if warnObj["rule"] != "WARN-01" {
		t.Errorf("warning rule = %v", warnObj["rule"])
	}
	if warnObj["severity"] != "warning" {
		t.Errorf("warning severity = %v", warnObj["severity"])
	}
	warnLoc := warnObj["location"].(map[string]any)
	if _, hasLine := warnLoc["line"]; hasLine {
		t.Error("warning location should omit line=0")
	}

	summary := m["summary"].(map[string]any)
	if summary["error_count"] != float64(1) {
		t.Errorf("error_count = %v", summary["error_count"])
	}
	if summary["warning_count"] != float64(1) {
		t.Errorf("warning_count = %v", summary["warning_count"])
	}
}

func TestFormatJSONRoundTrip(t *testing.T) {
	r := NewReport("test.json")
	r.SchemaValid = true
	r.AddFinding(NewError("RULE-03", "bad ref", Location{File: "test.json", Path: "$.rules[0]"}))

	data, err := FormatJSON(r)
	if err != nil {
		t.Fatalf("FormatJSON: %v", err)
	}

	// Verify it's valid JSON by unmarshalling
	var r2 Report
	if err := json.Unmarshal(data, &r2); err != nil {
		t.Fatalf("round-trip unmarshal: %v", err)
	}
	if r2.File != r.File {
		t.Errorf("file mismatch: %q vs %q", r2.File, r.File)
	}
	if len(r2.Errors) != 1 {
		t.Errorf("errors len = %d, want 1", len(r2.Errors))
	}
}

// Verify that the keys required by the spec are present.
func TestFormatJSONRequiredKeys(t *testing.T) {
	r := NewReport("x.json")
	data, err := FormatJSON(r)
	if err != nil {
		t.Fatalf("FormatJSON: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	required := []string{"file", "schema_valid", "errors", "warnings", "summary"}
	for _, key := range required {
		if _, ok := m[key]; !ok {
			t.Errorf("missing required key %q", key)
		}
	}
}
