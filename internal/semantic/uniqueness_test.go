package semantic

import (
	"testing"

	"github.com/foundry-zero/allium/internal/ast"
)

func TestCheckUniqueness_Clean(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Rules: []ast.Rule{
			{Name: "R1", Trigger: ast.Trigger{Kind: "external_stimulus", Name: "do_thing", Parameters: []ast.TriggerParam{{Name: "arg1"}}}},
			{Name: "R2", Trigger: ast.Trigger{Kind: "external_stimulus", Name: "do_thing", Parameters: []ast.TriggerParam{{Name: "arg1"}}}},
		},
		Given: []ast.GivenBinding{
			{Name: "a"},
			{Name: "b"},
		},
		Config: []ast.ConfigParam{
			{Name: "x"},
			{Name: "y"},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckUniqueness(spec, st)

	if len(findings) > 0 {
		for _, f := range findings {
			t.Errorf("unexpected: [%s] %s", f.Rule, f.Message)
		}
	}
}

func TestCheckUniqueness_RULE06_DifferentParamCount(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Rules: []ast.Rule{
			{Name: "R1", Trigger: ast.Trigger{Kind: "external_stimulus", Name: "trigger_x", Parameters: []ast.TriggerParam{{Name: "a"}}}},
			{Name: "R2", Trigger: ast.Trigger{Kind: "external_stimulus", Name: "trigger_x", Parameters: []ast.TriggerParam{{Name: "a"}, {Name: "b"}}}},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckUniqueness(spec, st)

	r06 := findingsWithRule(findings, "RULE-06")
	if len(r06) == 0 {
		t.Fatal("expected RULE-06 for incompatible param count")
	}
}

func TestCheckUniqueness_RULE06_DifferentParamNames(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Rules: []ast.Rule{
			{Name: "R1", Trigger: ast.Trigger{Kind: "external_stimulus", Name: "trigger_y", Parameters: []ast.TriggerParam{{Name: "owner"}}}},
			{Name: "R2", Trigger: ast.Trigger{Kind: "external_stimulus", Name: "trigger_y", Parameters: []ast.TriggerParam{{Name: "user"}}}},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckUniqueness(spec, st)

	r06 := findingsWithRule(findings, "RULE-06")
	if len(r06) == 0 {
		t.Fatal("expected RULE-06 for mismatched param names")
	}
}

func TestCheckUniqueness_RULE06_CompatibleParams(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Rules: []ast.Rule{
			{Name: "R1", Trigger: ast.Trigger{Kind: "external_stimulus", Name: "trigger_z", Parameters: []ast.TriggerParam{{Name: "a"}, {Name: "b"}}}},
			{Name: "R2", Trigger: ast.Trigger{Kind: "external_stimulus", Name: "trigger_z", Parameters: []ast.TriggerParam{{Name: "a"}, {Name: "b"}}}},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckUniqueness(spec, st)

	r06 := findingsWithRule(findings, "RULE-06")
	if len(r06) > 0 {
		t.Errorf("compatible params should not trigger RULE-06, got %d", len(r06))
	}
}

func TestCheckUniqueness_RULE06_SingleRule(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Rules: []ast.Rule{
			{Name: "R1", Trigger: ast.Trigger{Kind: "external_stimulus", Name: "solo"}},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckUniqueness(spec, st)

	r06 := findingsWithRule(findings, "RULE-06")
	if len(r06) > 0 {
		t.Error("single rule should never trigger RULE-06")
	}
}

func TestCheckUniqueness_RULE06_ChainedTrigger(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Rules: []ast.Rule{
			{Name: "R1", Trigger: ast.Trigger{Kind: "chained", Name: "after_x", Parameters: []ast.TriggerParam{{Name: "a"}}}},
			{Name: "R2", Trigger: ast.Trigger{Kind: "chained", Name: "after_x", Parameters: []ast.TriggerParam{{Name: "b"}}}},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckUniqueness(spec, st)

	r06 := findingsWithRule(findings, "RULE-06")
	if len(r06) == 0 {
		t.Fatal("expected RULE-06 for chained trigger with mismatched params")
	}
}

func TestCheckUniqueness_RULE23(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Given: []ast.GivenBinding{
			{Name: "current_user"},
			{Name: "current_user"},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckUniqueness(spec, st)

	r23 := findingsWithRule(findings, "RULE-23")
	if len(r23) != 1 {
		t.Fatalf("expected 1 RULE-23 finding, got %d", len(r23))
	}
}

func TestCheckUniqueness_RULE23_Unique(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Given: []ast.GivenBinding{
			{Name: "a"},
			{Name: "b"},
			{Name: "c"},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckUniqueness(spec, st)

	r23 := findingsWithRule(findings, "RULE-23")
	if len(r23) > 0 {
		t.Error("unique given names should not trigger RULE-23")
	}
}

func TestCheckUniqueness_RULE26(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Config: []ast.ConfigParam{
			{Name: "max_retries"},
			{Name: "timeout"},
			{Name: "max_retries"},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckUniqueness(spec, st)

	r26 := findingsWithRule(findings, "RULE-26")
	if len(r26) != 1 {
		t.Fatalf("expected 1 RULE-26 finding, got %d", len(r26))
	}
}

func TestCheckUniqueness_RULE26_Unique(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Config: []ast.ConfigParam{
			{Name: "x"},
			{Name: "y"},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckUniqueness(spec, st)

	r26 := findingsWithRule(findings, "RULE-26")
	if len(r26) > 0 {
		t.Error("unique config names should not trigger RULE-26")
	}
}

func TestCheckUniqueness_MultipleViolations(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Rules: []ast.Rule{
			{Name: "R1", Trigger: ast.Trigger{Kind: "external_stimulus", Name: "t", Parameters: []ast.TriggerParam{{Name: "a"}}}},
			{Name: "R2", Trigger: ast.Trigger{Kind: "external_stimulus", Name: "t", Parameters: []ast.TriggerParam{}}},
		},
		Given: []ast.GivenBinding{
			{Name: "dup"},
			{Name: "dup"},
		},
		Config: []ast.ConfigParam{
			{Name: "dup"},
			{Name: "dup"},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckUniqueness(spec, st)

	rules := map[string]int{}
	for _, f := range findings {
		rules[f.Rule]++
	}
	if rules["RULE-06"] == 0 {
		t.Error("missing RULE-06")
	}
	if rules["RULE-23"] == 0 {
		t.Error("missing RULE-23")
	}
	if rules["RULE-26"] == 0 {
		t.Error("missing RULE-26")
	}
}
