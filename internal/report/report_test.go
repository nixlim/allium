package report

import (
	"encoding/json"
	"testing"
)

func TestSeverityString(t *testing.T) {
	tests := []struct {
		sev  Severity
		want string
	}{
		{SeverityError, "error"},
		{SeverityWarning, "warning"},
		{Severity(99), "severity(99)"},
	}
	for _, tt := range tests {
		if got := tt.sev.String(); got != tt.want {
			t.Errorf("Severity(%d).String() = %q, want %q", tt.sev, got, tt.want)
		}
	}
}

func TestSeverityMarshalText(t *testing.T) {
	type wrapper struct {
		Sev Severity `json:"sev"`
	}
	w := wrapper{Sev: SeverityWarning}
	data, err := json.Marshal(w)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	want := `{"sev":"warning"}`
	if string(data) != want {
		t.Errorf("got %s, want %s", data, want)
	}
}

func TestNewFinding(t *testing.T) {
	loc := Location{File: "test.allium.json", Path: "$.entities[0]", Line: 10}
	f := NewFinding("RULE-01", SeverityError, "entity not found", loc)

	if f.Rule != "RULE-01" {
		t.Errorf("Rule = %q, want RULE-01", f.Rule)
	}
	if f.Severity != SeverityError {
		t.Errorf("Severity = %v, want SeverityError", f.Severity)
	}
	if f.Message != "entity not found" {
		t.Errorf("Message = %q, want %q", f.Message, "entity not found")
	}
	if f.Location != loc {
		t.Errorf("Location = %+v, want %+v", f.Location, loc)
	}
}

func TestNewErrorAndNewWarning(t *testing.T) {
	loc := Location{File: "f.json", Path: "$.rules[0]"}
	e := NewError("RULE-06", "duplicate trigger", loc)
	if e.Severity != SeverityError {
		t.Errorf("NewError severity = %v, want SeverityError", e.Severity)
	}

	w := NewWarning("WARN-01", "unused entity", loc)
	if w.Severity != SeverityWarning {
		t.Errorf("NewWarning severity = %v, want SeverityWarning", w.Severity)
	}
}

func TestReportAddFinding(t *testing.T) {
	r := NewReport("test.allium.json")

	if r.HasErrors() {
		t.Error("new report should not have errors")
	}
	if r.HasWarnings() {
		t.Error("new report should not have warnings")
	}

	loc := Location{File: "test.allium.json", Path: "$"}
	r.AddFinding(NewError("RULE-01", "bad ref", loc))
	r.AddFinding(NewError("RULE-03", "bad target", loc))
	r.AddFinding(NewWarning("WARN-01", "unused", loc))

	if r.Summary.ErrorCount != 2 {
		t.Errorf("ErrorCount = %d, want 2", r.Summary.ErrorCount)
	}
	if r.Summary.WarningCount != 1 {
		t.Errorf("WarningCount = %d, want 1", r.Summary.WarningCount)
	}
	if !r.HasErrors() {
		t.Error("report should have errors")
	}
	if !r.HasWarnings() {
		t.Error("report should have warnings")
	}
	if len(r.Errors) != 2 {
		t.Errorf("len(Errors) = %d, want 2", len(r.Errors))
	}
	if len(r.Warnings) != 1 {
		t.Errorf("len(Warnings) = %d, want 1", len(r.Warnings))
	}
}

func TestNewReportEmptySlices(t *testing.T) {
	r := NewReport("x.json")
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	// Errors and Warnings should be [] not null in JSON
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	for _, key := range []string{"errors", "warnings"} {
		v, ok := m[key]
		if !ok {
			t.Errorf("missing key %q in JSON", key)
			continue
		}
		arr, ok := v.([]any)
		if !ok {
			t.Errorf("key %q is not an array", key)
			continue
		}
		if len(arr) != 0 {
			t.Errorf("key %q has %d items, want 0", key, len(arr))
		}
	}
}

func TestLocationOmitsZeroLine(t *testing.T) {
	loc := Location{File: "f.json", Path: "$.entities[0]"}
	data, err := json.Marshal(loc)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if _, ok := m["line"]; ok {
		t.Error("line=0 should be omitted from JSON")
	}
}
