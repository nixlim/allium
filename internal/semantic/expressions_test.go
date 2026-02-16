package semantic

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/foundry-zero/allium/internal/ast"
)

// --- Test helpers ---

func intLitExpr(val int) *ast.Expression {
	raw, _ := json.Marshal(val)
	return &ast.Expression{Kind: "literal", Type: "integer", LitValue: raw}
}

func strLitExpr(val string) *ast.Expression {
	raw, _ := json.Marshal(val)
	return &ast.Expression{Kind: "literal", Type: "string", LitValue: raw}
}

func boolLitExpr(val bool) *ast.Expression {
	raw, _ := json.Marshal(val)
	return &ast.Expression{Kind: "literal", Type: "boolean", LitValue: raw}
}

func tsLitExpr(val string) *ast.Expression {
	raw, _ := json.Marshal(val)
	return &ast.Expression{Kind: "literal", Type: "timestamp", LitValue: raw}
}

func durLitExpr(val string) *ast.Expression {
	raw, _ := json.Marshal(val)
	return &ast.Expression{Kind: "literal", Type: "duration", LitValue: raw}
}

func enumLitExpr(val string) *ast.Expression {
	raw, _ := json.Marshal(val)
	return &ast.Expression{Kind: "literal", Type: "enum_value", LitValue: raw}
}

func comparisonExpr(op string, left, right *ast.Expression) *ast.Expression {
	return &ast.Expression{Kind: "comparison", Operator: op, Left: left, Right: right}
}

func arithmeticExpr(op string, left, right *ast.Expression) *ast.Expression {
	return &ast.Expression{Kind: "arithmetic", Operator: op, Left: left, Right: right}
}

// projectRoot returns the absolute path to the project root by finding go.mod.
func projectRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// --- Clean spec test ---

func TestCheckExpressions_Clean(t *testing.T) {
	spec := &ast.Spec{File: "test.allium.json"}
	st := BuildSymbolTable(spec)
	findings := CheckExpressions(spec, st)

	if len(findings) > 0 {
		for _, f := range findings {
			t.Errorf("unexpected: [%s] %s", f.Rule, f.Message)
		}
	}
}

// --- Acceptance criterion 1: password-auth.allium.json returns zero findings ---

func TestCheckExpressions_PasswordAuth_Clean(t *testing.T) {
	root := projectRoot()
	if root == "" {
		t.Skip("cannot locate project root")
	}
	path := filepath.Join(root, "schemas", "v1", "examples", "password-auth.allium.json")
	spec, err := ast.LoadSpec(path)
	if err != nil {
		t.Fatalf("failed to load password-auth spec: %v", err)
	}

	st := BuildSymbolTable(spec)
	findings := CheckExpressions(spec, st)

	if len(findings) > 0 {
		for _, f := range findings {
			t.Errorf("unexpected finding: [%s] %s at %s", f.Rule, f.Message, f.Location.Path)
		}
	}
}

// --- RULE-10: Derived value cycle detection ---

func TestCheckExpressions_RULE10_Cycle(t *testing.T) {
	// A references B, B references A
	spec := &ast.Spec{
		File: "test.allium.json",
		Entities: []ast.Entity{
			{
				Name: "Order",
				DerivedValues: []ast.DerivedValue{
					{Name: "total", Expression: &ast.Expression{Kind: "field_access", Field: "tax"}},
					{Name: "tax", Expression: &ast.Expression{Kind: "field_access", Field: "total"}},
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckExpressions(spec, st)

	r10 := findingsWithRule(findings, "RULE-10")
	if len(r10) == 0 {
		t.Fatal("expected RULE-10 for derived value cycle")
	}
	if !strings.Contains(r10[0].Message, "Cycle detected") {
		t.Errorf("message = %q", r10[0].Message)
	}
}

func TestCheckExpressions_RULE10_NoCycle(t *testing.T) {
	// A references B (no cycle)
	spec := &ast.Spec{
		File: "test.allium.json",
		Entities: []ast.Entity{
			{
				Name: "Order",
				DerivedValues: []ast.DerivedValue{
					{Name: "total", Expression: &ast.Expression{Kind: "field_access", Field: "tax"}},
					{Name: "tax", Expression: &ast.Expression{Kind: "literal", Type: "integer"}},
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckExpressions(spec, st)

	r10 := findingsWithRule(findings, "RULE-10")
	if len(r10) > 0 {
		t.Errorf("no cycle should be detected, got: %v", r10)
	}
}

func TestCheckExpressions_RULE10_ThreeWayCycle(t *testing.T) {
	// A -> B -> C -> A
	spec := &ast.Spec{
		File: "test.allium.json",
		Entities: []ast.Entity{
			{
				Name: "E",
				DerivedValues: []ast.DerivedValue{
					{Name: "a", Expression: &ast.Expression{Kind: "field_access", Field: "b"}},
					{Name: "b", Expression: &ast.Expression{Kind: "field_access", Field: "c"}},
					{Name: "c", Expression: &ast.Expression{Kind: "field_access", Field: "a"}},
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckExpressions(spec, st)

	r10 := findingsWithRule(findings, "RULE-10")
	if len(r10) == 0 {
		t.Fatal("expected RULE-10 for 3-way cycle")
	}
}

func TestCheckExpressions_RULE10_SingleDerived(t *testing.T) {
	// Only one derived value -- can't have a cycle
	spec := &ast.Spec{
		File: "test.allium.json",
		Entities: []ast.Entity{
			{
				Name: "E",
				DerivedValues: []ast.DerivedValue{
					{Name: "a", Expression: &ast.Expression{Kind: "literal"}},
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckExpressions(spec, st)

	r10 := findingsWithRule(findings, "RULE-10")
	if len(r10) > 0 {
		t.Error("single derived value should not trigger cycle detection")
	}
}

func TestCheckExpressions_RULE10_ValueTypeCycle(t *testing.T) {
	// Value type derived value cycle: X -> Y -> X
	spec := &ast.Spec{
		File: "test.allium.json",
		ValueTypes: []ast.ValueType{
			{
				Name: "Money",
				DerivedValues: []ast.DerivedValue{
					{Name: "x", Expression: &ast.Expression{Kind: "field_access", Field: "y"}},
					{Name: "y", Expression: &ast.Expression{Kind: "field_access", Field: "x"}},
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckExpressions(spec, st)

	r10 := findingsWithRule(findings, "RULE-10")
	if len(r10) == 0 {
		t.Fatal("expected RULE-10 for value type derived value cycle")
	}
}

// --- RULE-11: Out-of-scope field access ---

func TestCheckExpressions_RULE11_OutOfScope(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Rules: []ast.Rule{
			{
				Name:    "R1",
				Trigger: ast.Trigger{Kind: "external_stimulus", Name: "do_thing", Parameters: []ast.TriggerParam{{Name: "x"}}},
				Requires: []ast.Expression{
					{Kind: "field_access", Field: "unknown_var"}, // not in scope
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckExpressions(spec, st)

	r11 := findingsWithRule(findings, "RULE-11")
	if len(r11) == 0 {
		t.Fatal("expected RULE-11 for out-of-scope identifier")
	}
	if !strings.Contains(r11[0].Message, "unknown_var") {
		t.Errorf("message = %q, want mention of 'unknown_var'", r11[0].Message)
	}
}

func TestCheckExpressions_RULE11_TriggerParamsInScope(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Rules: []ast.Rule{
			{
				Name:    "R1",
				Trigger: ast.Trigger{Kind: "external_stimulus", Name: "do_thing", Parameters: []ast.TriggerParam{{Name: "x"}}},
				Requires: []ast.Expression{
					{Kind: "field_access", Field: "x"}, // in scope via trigger params
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckExpressions(spec, st)

	r11 := findingsWithRule(findings, "RULE-11")
	if len(r11) > 0 {
		t.Errorf("trigger param 'x' should be in scope, got: %v", r11)
	}
}

func TestCheckExpressions_RULE11_BindingInScope(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Rules: []ast.Rule{
			{
				Name:    "R1",
				Trigger: ast.Trigger{Kind: "state_transition", Entity: "Order", Binding: "order", Field: "status"},
				Requires: []ast.Expression{
					{Kind: "field_access", Field: "order"}, // in scope via trigger binding
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckExpressions(spec, st)

	r11 := findingsWithRule(findings, "RULE-11")
	if len(r11) > 0 {
		t.Errorf("binding 'order' should be in scope, got: %v", r11)
	}
}

func TestCheckExpressions_RULE11_GivenInScope(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Given: []ast.GivenBinding{
			{Name: "email_service", Type: ast.FieldType{Kind: "entity_ref", Entity: "EmailService"}},
		},
		Rules: []ast.Rule{
			{
				Name:    "R1",
				Trigger: ast.Trigger{Kind: "external_stimulus", Name: "test"},
				Requires: []ast.Expression{
					{Kind: "field_access", Field: "email_service"}, // in scope via given
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckExpressions(spec, st)

	r11 := findingsWithRule(findings, "RULE-11")
	if len(r11) > 0 {
		t.Errorf("given binding 'email_service' should be in scope, got: %v", r11)
	}
}

func TestCheckExpressions_RULE11_ConfigInScope(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Config: []ast.ConfigParam{
			{Name: "max_retries", Type: ast.FieldType{Kind: "primitive", Value: "Integer"}},
		},
		Rules: []ast.Rule{
			{
				Name:    "R1",
				Trigger: ast.Trigger{Kind: "external_stimulus", Name: "test"},
				Requires: []ast.Expression{
					// "config" is a magic root, config.max_retries is accessed via
					// field_access with object=field_access("config")
					{
						Kind:   "field_access",
						Object: &ast.Expression{Kind: "field_access", Field: "config"},
						Field:  "max_retries",
					},
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckExpressions(spec, st)

	r11 := findingsWithRule(findings, "RULE-11")
	if len(r11) > 0 {
		t.Errorf("'config' should be in scope, got: %v", r11)
	}
}

func TestCheckExpressions_RULE11_LetBindingInScope(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Rules: []ast.Rule{
			{
				Name:    "R1",
				Trigger: ast.Trigger{Kind: "external_stimulus", Name: "test", Parameters: []ast.TriggerParam{{Name: "x"}}},
				LetBindings: []ast.LetBinding{
					{Name: "y", Expression: &ast.Expression{Kind: "field_access", Field: "x"}},
				},
				Ensures: []ast.EnsuresClause{
					{
						Kind:   "entity_removal",
						Target: &ast.Expression{Kind: "field_access", Field: "y"}, // in scope via let binding
					},
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckExpressions(spec, st)

	r11 := findingsWithRule(findings, "RULE-11")
	if len(r11) > 0 {
		t.Errorf("let binding 'y' should be in scope, got: %v", r11)
	}
}

func TestCheckExpressions_RULE11_DefaultInScope(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Defaults: []ast.Default{
			{Entity: "User", Name: "system_user"},
		},
		Rules: []ast.Rule{
			{
				Name:    "R1",
				Trigger: ast.Trigger{Kind: "external_stimulus", Name: "test"},
				Requires: []ast.Expression{
					{Kind: "field_access", Field: "system_user"},
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckExpressions(spec, st)

	r11 := findingsWithRule(findings, "RULE-11")
	if len(r11) > 0 {
		t.Errorf("default name 'system_user' should be in scope, got: %v", r11)
	}
}

func TestCheckExpressions_RULE11_IterationBindingInScope(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Rules: []ast.Rule{
			{
				Name:    "R1",
				Trigger: ast.Trigger{Kind: "external_stimulus", Name: "test", Parameters: []ast.TriggerParam{{Name: "items"}}},
				Ensures: []ast.EnsuresClause{
					{
						Kind:       "iteration",
						Binding:    "item",
						Collection: &ast.Expression{Kind: "field_access", Field: "items"},
						Body: []ast.EnsuresClause{
							{
								Kind:   "entity_removal",
								Target: &ast.Expression{Kind: "field_access", Field: "item"},
							},
						},
					},
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckExpressions(spec, st)

	r11 := findingsWithRule(findings, "RULE-11")
	if len(r11) > 0 {
		t.Errorf("iteration binding 'item' should be in scope in body, got: %v", r11)
	}
}

func TestCheckExpressions_RULE11_ChainedFieldAccessNotChecked(t *testing.T) {
	// Chained field access (user.email) should not flag the inner field
	spec := &ast.Spec{
		File: "test.allium.json",
		Rules: []ast.Rule{
			{
				Name:    "R1",
				Trigger: ast.Trigger{Kind: "external_stimulus", Name: "test", Parameters: []ast.TriggerParam{{Name: "user"}}},
				Requires: []ast.Expression{
					{
						Kind:   "field_access",
						Object: &ast.Expression{Kind: "field_access", Field: "user"},
						Field:  "email",
					},
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckExpressions(spec, st)

	r11 := findingsWithRule(findings, "RULE-11")
	if len(r11) > 0 {
		t.Errorf("chained field access should not trigger RULE-11, got: %v", r11)
	}
}

func TestCheckExpressions_RULE11_ForClauseBindingInScope(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Rules: []ast.Rule{
			{
				Name:    "R1",
				Trigger: ast.Trigger{Kind: "external_stimulus", Name: "test", Parameters: []ast.TriggerParam{{Name: "items"}}},
				ForClause: &ast.ForClause{
					Binding:    "item",
					Collection: &ast.Expression{Kind: "field_access", Field: "items"},
				},
				Ensures: []ast.EnsuresClause{
					{
						Kind:   "entity_removal",
						Target: &ast.Expression{Kind: "field_access", Field: "item"},
					},
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckExpressions(spec, st)

	r11 := findingsWithRule(findings, "RULE-11")
	if len(r11) > 0 {
		t.Errorf("for_clause binding 'item' should be in scope for ensures, got: %v", r11)
	}
}

// --- RULE-12: Type mismatch checks ---

func TestCheckExpressions_RULE12_IntegerVsString(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Rules: []ast.Rule{
			{
				Name:    "R1",
				Trigger: ast.Trigger{Kind: "external_stimulus", Name: "test"},
				Requires: []ast.Expression{
					*comparisonExpr("=", intLitExpr(42), strLitExpr("hello")),
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckExpressions(spec, st)

	r12 := findingsWithRule(findings, "RULE-12")
	if len(r12) == 0 {
		t.Fatal("expected RULE-12 for Integer vs String comparison")
	}
	if !strings.Contains(r12[0].Message, "Type mismatch") {
		t.Errorf("message = %q", r12[0].Message)
	}
}

func TestCheckExpressions_RULE12_BooleanPlusInteger(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Rules: []ast.Rule{
			{
				Name:    "R1",
				Trigger: ast.Trigger{Kind: "external_stimulus", Name: "test"},
				Requires: []ast.Expression{
					*arithmeticExpr("+", boolLitExpr(true), intLitExpr(1)),
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckExpressions(spec, st)

	r12 := findingsWithRule(findings, "RULE-12")
	if len(r12) == 0 {
		t.Fatal("expected RULE-12 for Boolean + Integer arithmetic")
	}
	if !strings.Contains(r12[0].Message, "Non-numeric type Boolean") {
		t.Errorf("message = %q, want 'Non-numeric type Boolean'", r12[0].Message)
	}
}

func TestCheckExpressions_RULE12_TimestampMinusDuration_Valid(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Rules: []ast.Rule{
			{
				Name:    "R1",
				Trigger: ast.Trigger{Kind: "external_stimulus", Name: "test"},
				Requires: []ast.Expression{
					// Timestamp - Duration is valid temporal arithmetic
					*arithmeticExpr("-", tsLitExpr("now"), durLitExpr("5.minutes")),
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckExpressions(spec, st)

	r12 := findingsWithRule(findings, "RULE-12")
	if len(r12) > 0 {
		t.Errorf("Timestamp - Duration should be valid, got: %v", r12)
	}
}

func TestCheckExpressions_RULE12_TimestampPlusDuration_Valid(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Rules: []ast.Rule{
			{
				Name:    "R1",
				Trigger: ast.Trigger{Kind: "external_stimulus", Name: "test"},
				Requires: []ast.Expression{
					*arithmeticExpr("+", tsLitExpr("now"), durLitExpr("1.hour")),
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckExpressions(spec, st)

	r12 := findingsWithRule(findings, "RULE-12")
	if len(r12) > 0 {
		t.Errorf("Timestamp + Duration should be valid, got: %v", r12)
	}
}

func TestCheckExpressions_RULE12_TimestampMinusTimestamp_Valid(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Rules: []ast.Rule{
			{
				Name:    "R1",
				Trigger: ast.Trigger{Kind: "external_stimulus", Name: "test"},
				Requires: []ast.Expression{
					*arithmeticExpr("-", tsLitExpr("now"), tsLitExpr("yesterday")),
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckExpressions(spec, st)

	r12 := findingsWithRule(findings, "RULE-12")
	if len(r12) > 0 {
		t.Errorf("Timestamp - Timestamp should be valid, got: %v", r12)
	}
}

func TestCheckExpressions_RULE12_DurationPlusDuration_Valid(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Rules: []ast.Rule{
			{
				Name:    "R1",
				Trigger: ast.Trigger{Kind: "external_stimulus", Name: "test"},
				Requires: []ast.Expression{
					*arithmeticExpr("+", durLitExpr("5.minutes"), durLitExpr("10.minutes")),
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckExpressions(spec, st)

	r12 := findingsWithRule(findings, "RULE-12")
	if len(r12) > 0 {
		t.Errorf("Duration + Duration should be valid, got: %v", r12)
	}
}

func TestCheckExpressions_RULE12_IntegerTimesInteger_Valid(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Rules: []ast.Rule{
			{
				Name:    "R1",
				Trigger: ast.Trigger{Kind: "external_stimulus", Name: "test"},
				Requires: []ast.Expression{
					*arithmeticExpr("*", intLitExpr(2), intLitExpr(3)),
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckExpressions(spec, st)

	r12 := findingsWithRule(findings, "RULE-12")
	if len(r12) > 0 {
		t.Errorf("Integer * Integer should be valid, got: %v", r12)
	}
}

func TestCheckExpressions_RULE12_StringPlusString_Invalid(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Rules: []ast.Rule{
			{
				Name:    "R1",
				Trigger: ast.Trigger{Kind: "external_stimulus", Name: "test"},
				Requires: []ast.Expression{
					*arithmeticExpr("+", strLitExpr("a"), strLitExpr("b")),
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckExpressions(spec, st)

	r12 := findingsWithRule(findings, "RULE-12")
	if len(r12) == 0 {
		t.Fatal("expected RULE-12 for String + String arithmetic")
	}
}

func TestCheckExpressions_RULE12_CompareIntegerVsInteger_Valid(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Rules: []ast.Rule{
			{
				Name:    "R1",
				Trigger: ast.Trigger{Kind: "external_stimulus", Name: "test"},
				Requires: []ast.Expression{
					*comparisonExpr(">=", intLitExpr(10), intLitExpr(5)),
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckExpressions(spec, st)

	r12 := findingsWithRule(findings, "RULE-12")
	if len(r12) > 0 {
		t.Errorf("Integer >= Integer should be valid, got: %v", r12)
	}
}

func TestCheckExpressions_RULE12_CompareTimestampVsTimestamp_Valid(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Rules: []ast.Rule{
			{
				Name:    "R1",
				Trigger: ast.Trigger{Kind: "external_stimulus", Name: "test"},
				Requires: []ast.Expression{
					*comparisonExpr(">", tsLitExpr("now"), tsLitExpr("yesterday")),
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckExpressions(spec, st)

	r12 := findingsWithRule(findings, "RULE-12")
	if len(r12) > 0 {
		t.Errorf("Timestamp > Timestamp should be valid, got: %v", r12)
	}
}

func TestCheckExpressions_RULE12_BooleanVsInteger_Comparison(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Rules: []ast.Rule{
			{
				Name:    "R1",
				Trigger: ast.Trigger{Kind: "external_stimulus", Name: "test"},
				Requires: []ast.Expression{
					*comparisonExpr("=", boolLitExpr(true), intLitExpr(1)),
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckExpressions(spec, st)

	r12 := findingsWithRule(findings, "RULE-12")
	if len(r12) == 0 {
		t.Fatal("expected RULE-12 for Boolean vs Integer comparison")
	}
}

func TestCheckExpressions_RULE12_NullComparison_Valid(t *testing.T) {
	// Comparing anything to null is valid (null checks)
	spec := &ast.Spec{
		File: "test.allium.json",
		Rules: []ast.Rule{
			{
				Name:    "R1",
				Trigger: ast.Trigger{Kind: "external_stimulus", Name: "test"},
				Requires: []ast.Expression{
					*comparisonExpr("=", intLitExpr(42), &ast.Expression{Kind: "literal", Type: "null"}),
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckExpressions(spec, st)

	r12 := findingsWithRule(findings, "RULE-12")
	if len(r12) > 0 {
		t.Errorf("Integer = null comparison should be valid, got: %v", r12)
	}
}

func TestCheckExpressions_RULE12_EnumValueComparison_Valid(t *testing.T) {
	// Comparing enum_value literal with a field is valid
	spec := &ast.Spec{
		File: "test.allium.json",
		Rules: []ast.Rule{
			{
				Name:    "R1",
				Trigger: ast.Trigger{Kind: "external_stimulus", Name: "test"},
				Requires: []ast.Expression{
					*comparisonExpr("=",
						&ast.Expression{Kind: "field_access", Object: &ast.Expression{Kind: "field_access", Field: "user"}, Field: "status"},
						enumLitExpr("active"),
					),
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckExpressions(spec, st)

	r12 := findingsWithRule(findings, "RULE-12")
	if len(r12) > 0 {
		t.Errorf("field_access = enum_value should be valid, got: %v", r12)
	}
}

func TestCheckExpressions_RULE12_FieldTypeArithmetic(t *testing.T) {
	// Test that field types are resolved for arithmetic checks
	spec := &ast.Spec{
		File: "test.allium.json",
		Entities: []ast.Entity{
			{
				Name: "Order",
				Fields: []ast.Field{
					{Name: "amount", Type: ast.FieldType{Kind: "primitive", Value: "Integer"}},
					{Name: "name", Type: ast.FieldType{Kind: "primitive", Value: "String"}},
				},
				DerivedValues: []ast.DerivedValue{
					{
						Name: "bad_calc",
						Expression: arithmeticExpr("+",
							&ast.Expression{Kind: "field_access", Field: "amount"},
							&ast.Expression{Kind: "field_access", Field: "name"},
						),
					},
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckExpressions(spec, st)

	r12 := findingsWithRule(findings, "RULE-12")
	if len(r12) == 0 {
		t.Fatal("expected RULE-12 for Integer + String field arithmetic")
	}
}

// --- RULE-13: any/all lambda checks ---

func TestCheckExpressions_RULE13_MissingLambda(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Entities: []ast.Entity{
			{
				Name: "E",
				DerivedValues: []ast.DerivedValue{
					{
						Name: "count",
						Expression: &ast.Expression{
							Kind:      "collection_op",
							Operation: "any",
							Lambda:    nil, // missing lambda
						},
					},
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckExpressions(spec, st)

	r13 := findingsWithRule(findings, "RULE-13")
	if len(r13) == 0 {
		t.Fatal("expected RULE-13 for any without lambda")
	}
}

func TestCheckExpressions_RULE13_EmptyParameter(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Rules: []ast.Rule{
			{
				Name:    "R1",
				Trigger: ast.Trigger{Kind: "external_stimulus", Name: "test"},
				Requires: []ast.Expression{
					{
						Kind:      "collection_op",
						Operation: "all",
						Lambda:    &ast.Expression{Kind: "lambda", Parameter: ""},
					},
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckExpressions(spec, st)

	r13 := findingsWithRule(findings, "RULE-13")
	if len(r13) == 0 {
		t.Fatal("expected RULE-13 for all with empty lambda parameter")
	}
}

func TestCheckExpressions_RULE13_ValidLambda(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Rules: []ast.Rule{
			{
				Name:    "R1",
				Trigger: ast.Trigger{Kind: "external_stimulus", Name: "test"},
				Requires: []ast.Expression{
					{
						Kind:      "collection_op",
						Operation: "any",
						Lambda:    &ast.Expression{Kind: "lambda", Parameter: "item"},
					},
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckExpressions(spec, st)

	r13 := findingsWithRule(findings, "RULE-13")
	if len(r13) > 0 {
		t.Error("valid lambda should not trigger RULE-13")
	}
}

func TestCheckExpressions_RULE13_OtherCollectionOp(t *testing.T) {
	// "count" operation doesn't need a lambda
	spec := &ast.Spec{
		File: "test.allium.json",
		Rules: []ast.Rule{
			{
				Name:    "R1",
				Trigger: ast.Trigger{Kind: "external_stimulus", Name: "test"},
				Requires: []ast.Expression{
					{
						Kind:      "collection_op",
						Operation: "count",
					},
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckExpressions(spec, st)

	r13 := findingsWithRule(findings, "RULE-13")
	if len(r13) > 0 {
		t.Error("count operation should not trigger RULE-13")
	}
}

func TestCheckExpressions_RULE13_InEnsures(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Rules: []ast.Rule{
			{
				Name:    "R1",
				Trigger: ast.Trigger{Kind: "external_stimulus", Name: "test"},
				Ensures: []ast.EnsuresClause{
					{
						Kind: "conditional",
						Condition: &ast.Expression{
							Kind:      "collection_op",
							Operation: "all",
							Lambda:    nil,
						},
					},
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckExpressions(spec, st)

	r13 := findingsWithRule(findings, "RULE-13")
	if len(r13) == 0 {
		t.Fatal("expected RULE-13 for all in ensures condition without lambda")
	}
}

// --- RULE-14: Enum comparison checks ---

func TestCheckExpressions_RULE14_InlineEnumComparison(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Entities: []ast.Entity{
			{
				Name: "Order",
				Fields: []ast.Field{
					{Name: "status", Type: ast.FieldType{Kind: "inline_enum", Values: []string{"pending", "active"}}},
					{Name: "priority", Type: ast.FieldType{Kind: "inline_enum", Values: []string{"low", "high"}}},
				},
				DerivedValues: []ast.DerivedValue{
					{
						Name: "check",
						Expression: &ast.Expression{
							Kind:     "comparison",
							Operator: "=",
							Left:     &ast.Expression{Kind: "field_access", Field: "status"},
							Right:    &ast.Expression{Kind: "field_access", Field: "priority"},
						},
					},
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckExpressions(spec, st)

	r14 := findingsWithRule(findings, "RULE-14")
	if len(r14) == 0 {
		t.Fatal("expected RULE-14 for inline enum cross-comparison")
	}
}

func TestCheckExpressions_RULE14_DifferentNamedEnums(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Enumerations: []ast.Enumeration{
			{Name: "Priority", Values: []string{"low", "high"}},
			{Name: "Status", Values: []string{"active", "inactive"}},
		},
		Entities: []ast.Entity{
			{
				Name: "Task",
				Fields: []ast.Field{
					{Name: "priority", Type: ast.FieldType{Kind: "named_enum", Name: "Priority"}},
					{Name: "status", Type: ast.FieldType{Kind: "named_enum", Name: "Status"}},
				},
			},
		},
		Rules: []ast.Rule{
			{
				Name:    "R1",
				Trigger: ast.Trigger{Kind: "state_transition", Entity: "Task", Field: "status", Binding: "task"},
				Requires: []ast.Expression{
					{
						Kind:     "comparison",
						Operator: "=",
						Left:     &ast.Expression{Kind: "field_access", Field: "priority"},
						Right:    &ast.Expression{Kind: "field_access", Field: "status"},
					},
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckExpressions(spec, st)

	r14 := findingsWithRule(findings, "RULE-14")
	if len(r14) == 0 {
		t.Fatal("expected RULE-14 for different named enum comparison")
	}
}

func TestCheckExpressions_RULE14_SameNamedEnum(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Enumerations: []ast.Enumeration{
			{Name: "Priority", Values: []string{"low", "high"}},
		},
		Entities: []ast.Entity{
			{
				Name: "Task",
				Fields: []ast.Field{
					{Name: "priority", Type: ast.FieldType{Kind: "named_enum", Name: "Priority"}},
					{Name: "urgency", Type: ast.FieldType{Kind: "named_enum", Name: "Priority"}},
				},
			},
		},
		Rules: []ast.Rule{
			{
				Name:    "R1",
				Trigger: ast.Trigger{Kind: "state_transition", Entity: "Task", Field: "priority", Binding: "task"},
				Requires: []ast.Expression{
					{
						Kind:     "comparison",
						Operator: "=",
						Left:     &ast.Expression{Kind: "field_access", Field: "priority"},
						Right:    &ast.Expression{Kind: "field_access", Field: "urgency"},
					},
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckExpressions(spec, st)

	r14 := findingsWithRule(findings, "RULE-14")
	if len(r14) > 0 {
		t.Error("same named enum comparison should not trigger RULE-14")
	}
}

// --- Tarjan SCC unit tests ---

func TestTarjanSCC_NoCycle(t *testing.T) {
	// 0 -> 1 -> 2 (linear, no cycle)
	adj := [][]int{{1}, {2}, {}}
	sccs := tarjanSCC(adj)

	for _, scc := range sccs {
		if len(scc) > 1 {
			t.Errorf("expected no multi-node SCC, got %v", scc)
		}
	}
}

func TestTarjanSCC_SimpleCycle(t *testing.T) {
	// 0 -> 1 -> 0
	adj := [][]int{{1}, {0}}
	sccs := tarjanSCC(adj)

	found := false
	for _, scc := range sccs {
		if len(scc) == 2 {
			found = true
		}
	}
	if !found {
		t.Error("expected a 2-node SCC")
	}
}

func TestTarjanSCC_ThreeCycle(t *testing.T) {
	// 0 -> 1 -> 2 -> 0
	adj := [][]int{{1}, {2}, {0}}
	sccs := tarjanSCC(adj)

	found := false
	for _, scc := range sccs {
		if len(scc) == 3 {
			found = true
		}
	}
	if !found {
		t.Error("expected a 3-node SCC")
	}
}

func TestTarjanSCC_Empty(t *testing.T) {
	sccs := tarjanSCC(nil)
	if len(sccs) != 0 {
		t.Errorf("expected empty, got %v", sccs)
	}
}
