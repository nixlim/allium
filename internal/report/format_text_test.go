package report

import (
	"strings"
	"testing"
)

func TestFormatTextEmpty(t *testing.T) {
	r := NewReport("clean.allium.json")
	r.SchemaValid = true
	out := FormatText(r)

	if !strings.Contains(out, "File: clean.allium.json") {
		t.Error("output should contain file name")
	}
	if !strings.Contains(out, "0 errors, 0 warnings") {
		t.Errorf("expected zero summary, got:\n%s", out)
	}
}

func TestFormatTextWithFindings(t *testing.T) {
	r := NewReport("bad.allium.json")
	r.SchemaValid = false
	r.AddFinding(NewError("RULE-01", "Entity 'Foo' not declared", Location{
		File: "bad.allium.json",
		Path: "$.entities[0].fields[1].type",
		Line: 42,
	}))
	r.AddFinding(NewWarning("WARN-03", "Unreachable state 'dormant'", Location{
		File: "bad.allium.json",
		Path: "$.entities[0].lifecycle.states[2]",
	}))

	out := FormatText(r)

	// Check errors appear before warnings
	errIdx := strings.Index(out, "RULE-01")
	warnIdx := strings.Index(out, "WARN-03")
	if errIdx < 0 || warnIdx < 0 {
		t.Fatalf("missing rule IDs in output:\n%s", out)
	}
	if errIdx > warnIdx {
		t.Error("errors should appear before warnings")
	}

	// Check content
	if !strings.Contains(out, "[RULE-01] error: Entity 'Foo' not declared") {
		t.Errorf("error finding not formatted correctly:\n%s", out)
	}
	if !strings.Contains(out, "(line 42)") {
		t.Errorf("line number missing:\n%s", out)
	}
	if !strings.Contains(out, "[WARN-03] warning: Unreachable state 'dormant'") {
		t.Errorf("warning finding not formatted correctly:\n%s", out)
	}
	if !strings.Contains(out, "1 errors, 1 warnings") {
		t.Errorf("summary wrong:\n%s", out)
	}
}

func TestFormatTextNoLineNumber(t *testing.T) {
	r := NewReport("test.json")
	r.AddFinding(NewError("RULE-06", "dup trigger", Location{
		File: "test.json",
		Path: "$.rules[0]",
	}))

	out := FormatText(r)
	if strings.Contains(out, "line") {
		t.Errorf("should not contain line reference when Line=0:\n%s", out)
	}
	if !strings.Contains(out, "at $.rules[0]") {
		t.Errorf("should show path:\n%s", out)
	}
}
