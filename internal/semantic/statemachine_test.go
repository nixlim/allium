package semantic

import (
	"encoding/json"
	"testing"

	"github.com/foundry-zero/allium/internal/ast"
)

func litExpr(val string) ast.Expression {
	raw, _ := json.Marshal(val)
	return ast.Expression{Kind: "literal", Type: "string", LitValue: raw}
}

func rawExpr(val string) json.RawMessage {
	expr := litExpr(val)
	data, _ := json.Marshal(expr)
	return data
}

func fieldAccess(field string) *ast.Expression {
	return &ast.Expression{Kind: "field_access", Field: field}
}

// makeStateMachineSpec creates a spec with a valid state machine:
// Order entity with status enum (pending, active, done),
// creation sets pending, transitions pending->active->done.
func makeStateMachineSpec() *ast.Spec {
	statusExpr := litExpr("pending")
	return &ast.Spec{
		File:    "test.allium.json",
		Version: "0.4.0",
		Entities: []ast.Entity{
			{
				Name: "Order",
				Fields: []ast.Field{
					{Name: "status", Type: ast.FieldType{Kind: "named_enum", Name: "OrderStatus"}},
					{Name: "amount", Type: ast.FieldType{Kind: "primitive", Value: "Integer"}},
				},
			},
		},
		Enumerations: []ast.Enumeration{
			{Name: "OrderStatus", Values: []string{"pending", "active", "done"}},
		},
		Rules: []ast.Rule{
			{
				Name:    "CreateOrder",
				Trigger: ast.Trigger{Kind: "external_stimulus", Name: "create_order"},
				Ensures: []ast.EnsuresClause{
					{
						Kind:   "entity_creation",
						Entity: "Order",
						Fields: map[string]ast.Expression{"status": statusExpr},
					},
				},
			},
			{
				Name:    "ActivateOrder",
				Trigger: ast.Trigger{Kind: "state_transition", Entity: "Order", Field: "status", Binding: "order"},
				Ensures: []ast.EnsuresClause{
					{
						Kind:   "state_change",
						Target: fieldAccess("status"),
						Value:  rawExpr("active"),
					},
				},
			},
			{
				Name:    "CompleteOrder",
				Trigger: ast.Trigger{Kind: "state_transition", Entity: "Order", Field: "status", Binding: "order"},
				Ensures: []ast.EnsuresClause{
					{
						Kind:   "state_change",
						Target: fieldAccess("status"),
						Value:  rawExpr("done"),
					},
				},
			},
		},
	}
}

func TestCheckStateMachines_Clean(t *testing.T) {
	spec := makeStateMachineSpec()
	st := BuildSymbolTable(spec)
	findings := CheckStateMachines(spec, st)

	if len(findings) > 0 {
		for _, f := range findings {
			t.Errorf("unexpected: [%s] %s", f.Rule, f.Message)
		}
	}
}

func TestCheckStateMachines_NoEnumFields(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Entities: []ast.Entity{
			{Name: "Simple", Fields: []ast.Field{
				{Name: "name", Type: ast.FieldType{Kind: "primitive", Value: "String"}},
			}},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckStateMachines(spec, st)

	if len(findings) > 0 {
		t.Errorf("no enum fields should produce no findings, got %d", len(findings))
	}
}

func TestCheckStateMachines_RULE07_Unreachable(t *testing.T) {
	spec := makeStateMachineSpec()
	// Add "archived" to enum but no transitions reach it
	spec.Enumerations[0].Values = []string{"pending", "active", "done", "archived"}
	st := BuildSymbolTable(spec)
	findings := CheckStateMachines(spec, st)

	r07 := findingsWithRule(findings, "RULE-07")
	if len(r07) == 0 {
		t.Fatal("expected RULE-07 for unreachable 'archived'")
	}
	found := false
	for _, f := range r07 {
		if f.Message == "Unreachable status value 'archived' on 'Order'" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected message about 'archived', got: %v", r07)
	}
}

func TestCheckStateMachines_RULE08_DeadEnd(t *testing.T) {
	// Create: open -> blocked (no transitions out of blocked)
	spec := &ast.Spec{
		File: "test.allium.json",
		Entities: []ast.Entity{
			{Name: "Task", Fields: []ast.Field{
				{Name: "status", Type: ast.FieldType{Kind: "inline_enum", Values: []string{"open", "blocked", "done"}}},
			}},
		},
		Rules: []ast.Rule{
			{
				Name:    "CreateTask",
				Trigger: ast.Trigger{Kind: "external_stimulus", Name: "create_task"},
				Ensures: []ast.EnsuresClause{
					{
						Kind:   "entity_creation",
						Entity: "Task",
						Fields: map[string]ast.Expression{"status": litExpr("open")},
					},
				},
			},
			{
				Name:    "BlockTask",
				Trigger: ast.Trigger{Kind: "state_transition", Entity: "Task", Field: "status", Binding: "task"},
				Ensures: []ast.EnsuresClause{
					{
						Kind:   "state_change",
						Target: fieldAccess("status"),
						Value:  rawExpr("blocked"),
					},
				},
			},
			// done is also reachable (conservative approach), but blocked has no exit
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckStateMachines(spec, st)

	r08 := findingsWithRule(findings, "RULE-08")
	if len(r08) == 0 {
		t.Fatal("expected RULE-08 for dead-end 'blocked'")
	}
}

func TestCheckStateMachines_RULE09_UndeclaredValue(t *testing.T) {
	spec := makeStateMachineSpec()
	// Change a transition to assign "cancelled" which isn't in the enum
	spec.Rules[1].Ensures[0].Value = rawExpr("cancelled")
	st := BuildSymbolTable(spec)
	findings := CheckStateMachines(spec, st)

	r09 := findingsWithRule(findings, "RULE-09")
	if len(r09) == 0 {
		t.Fatal("expected RULE-09 for undeclared value 'cancelled'")
	}
}

func TestCheckStateMachines_RULE09_UndeclaredCreationValue(t *testing.T) {
	spec := makeStateMachineSpec()
	// Change creation to use undeclared value
	spec.Rules[0].Ensures[0].Fields["status"] = litExpr("new")
	st := BuildSymbolTable(spec)
	findings := CheckStateMachines(spec, st)

	r09 := findingsWithRule(findings, "RULE-09")
	if len(r09) == 0 {
		t.Fatal("expected RULE-09 for undeclared creation value 'new'")
	}
}

func TestCheckStateMachines_ConditionalEnsures(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Entities: []ast.Entity{
			{Name: "Item", Fields: []ast.Field{
				{Name: "state", Type: ast.FieldType{Kind: "inline_enum", Values: []string{"a", "b"}}},
			}},
		},
		Rules: []ast.Rule{
			{
				Name:    "CreateItem",
				Trigger: ast.Trigger{Kind: "external_stimulus", Name: "create_item"},
				Ensures: []ast.EnsuresClause{
					{Kind: "entity_creation", Entity: "Item", Fields: map[string]ast.Expression{"state": litExpr("a")}},
				},
			},
			{
				Name:    "UpdateItem",
				Trigger: ast.Trigger{Kind: "state_transition", Entity: "Item", Field: "state", Binding: "item"},
				Ensures: []ast.EnsuresClause{
					{
						Kind: "conditional",
						Then: []ast.EnsuresClause{
							{Kind: "state_change", Target: fieldAccess("state"), Value: rawExpr("b")},
						},
					},
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckStateMachines(spec, st)

	// Should be clean — a->b via conditional, b is terminal (dead-end)
	r09 := findingsWithRule(findings, "RULE-09")
	if len(r09) > 0 {
		t.Errorf("expected no RULE-09, got %v", r09)
	}
}

func TestCheckStateMachines_InlineEnum(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Entities: []ast.Entity{
			{Name: "Ticket", Fields: []ast.Field{
				{Name: "priority", Type: ast.FieldType{Kind: "inline_enum", Values: []string{"low", "medium", "high"}}},
			}},
		},
		Rules: []ast.Rule{
			{
				Name:    "CreateTicket",
				Trigger: ast.Trigger{Kind: "external_stimulus", Name: "create_ticket"},
				Ensures: []ast.EnsuresClause{
					{Kind: "entity_creation", Entity: "Ticket", Fields: map[string]ast.Expression{"priority": litExpr("low")}},
				},
			},
			{
				Name:    "Escalate",
				Trigger: ast.Trigger{Kind: "state_transition", Entity: "Ticket", Field: "priority", Binding: "ticket"},
				Ensures: []ast.EnsuresClause{
					{Kind: "state_change", Target: fieldAccess("priority"), Value: rawExpr("bogus")},
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckStateMachines(spec, st)

	r09 := findingsWithRule(findings, "RULE-09")
	if len(r09) == 0 {
		t.Fatal("expected RULE-09 for undeclared inline enum value 'bogus'")
	}
}

// TestCheckStateMachines_CrossEntityFieldName verifies that a state_change on
// Session.status does not falsely affect User.status analysis when both entities
// share a "status" field name.
func TestCheckStateMachines_CrossEntityFieldName(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Entities: []ast.Entity{
			{
				Name: "User",
				Fields: []ast.Field{
					{Name: "status", Type: ast.FieldType{Kind: "named_enum", Name: "UserStatus"}},
				},
			},
			{
				Name: "Session",
				Fields: []ast.Field{
					{Name: "status", Type: ast.FieldType{Kind: "named_enum", Name: "SessionStatus"}},
				},
			},
		},
		Enumerations: []ast.Enumeration{
			{Name: "UserStatus", Values: []string{"active", "locked"}},
			{Name: "SessionStatus", Values: []string{"active", "revoked"}},
		},
		Rules: []ast.Rule{
			{
				Name:    "CreateUser",
				Trigger: ast.Trigger{Kind: "external_stimulus", Name: "register"},
				Ensures: []ast.EnsuresClause{
					{Kind: "entity_creation", Entity: "User", Fields: map[string]ast.Expression{"status": litExpr("active")}},
				},
			},
			{
				Name:    "LockUser",
				Trigger: ast.Trigger{Kind: "state_transition", Entity: "User", Field: "status", Binding: "user"},
				Ensures: []ast.EnsuresClause{
					{Kind: "state_change", Target: fieldAccess("status"), Value: rawExpr("locked")},
				},
			},
			{
				Name:    "CreateSession",
				Trigger: ast.Trigger{Kind: "external_stimulus", Name: "login"},
				Ensures: []ast.EnsuresClause{
					{Kind: "entity_creation", Entity: "Session", Fields: map[string]ast.Expression{"status": litExpr("active")}},
				},
			},
			{
				// This rule sets session.status — must NOT affect User.status analysis
				Name:    "Logout",
				Trigger: ast.Trigger{Kind: "state_transition", Entity: "Session", Field: "status", Binding: "session"},
				Ensures: []ast.EnsuresClause{
					{Kind: "state_change", Target: fieldAccess("status"), Value: rawExpr("revoked")},
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckStateMachines(spec, st)

	// "revoked" is valid for SessionStatus but NOT for UserStatus.
	// Before the fix, the Logout rule's state_change would falsely match
	// against User.status, triggering RULE-09 for "revoked" on User.
	r09 := findingsWithRule(findings, "RULE-09")
	for _, f := range r09 {
		if f.Message == "Undeclared status value 'revoked' assigned to 'User.status'" {
			t.Error("Logout rule's session.status change should not affect User.status analysis")
		}
	}
	if len(r09) > 0 {
		t.Errorf("expected no RULE-09 findings, got: %v", r09)
	}
}

// TestCheckStateMachines_ChainedFieldAccess verifies that chained field access
// (e.g., session.status) correctly matches when the binding resolves to the entity.
func TestCheckStateMachines_ChainedFieldAccess(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Entities: []ast.Entity{
			{
				Name: "Session",
				Fields: []ast.Field{
					{Name: "status", Type: ast.FieldType{Kind: "inline_enum", Values: []string{"active", "expired"}}},
				},
			},
		},
		Rules: []ast.Rule{
			{
				Name:    "CreateSession",
				Trigger: ast.Trigger{Kind: "external_stimulus", Name: "login"},
				Ensures: []ast.EnsuresClause{
					{Kind: "entity_creation", Entity: "Session", Fields: map[string]ast.Expression{"status": litExpr("active")}},
				},
			},
			{
				Name:    "ExpireSession",
				Trigger: ast.Trigger{Kind: "temporal", Entity: "Session", Binding: "session"},
				Ensures: []ast.EnsuresClause{
					{
						Kind: "state_change",
						Target: &ast.Expression{
							Kind:   "field_access",
							Field:  "status",
							Object: &ast.Expression{Kind: "field_access", Field: "session"},
						},
						Value: rawExpr("expired"),
					},
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckStateMachines(spec, st)

	// The chained access "session.status" should match Session.status
	// because "session" binding resolves to entity "Session"
	r09 := findingsWithRule(findings, "RULE-09")
	if len(r09) > 0 {
		t.Errorf("expected no RULE-09 for valid chained field access, got: %v", r09)
	}

	// Both "active" and "expired" should be reachable
	r07 := findingsWithRule(findings, "RULE-07")
	if len(r07) > 0 {
		t.Errorf("expected no RULE-07 (all values reachable), got: %v", r07)
	}
}

func TestBfsReachable(t *testing.T) {
	transitions := map[string][]string{
		"a": {"b"},
		"b": {"c"},
		"c": {"d"},
	}
	reachable := bfsReachable([]string{"a"}, transitions)

	for _, v := range []string{"a", "b", "c", "d"} {
		if !reachable[v] {
			t.Errorf("%s should be reachable", v)
		}
	}
	if reachable["e"] {
		t.Error("e should not be reachable")
	}
}

func TestBfsReachable_Cycle(t *testing.T) {
	transitions := map[string][]string{
		"a": {"b"},
		"b": {"a"},
	}
	reachable := bfsReachable([]string{"a"}, transitions)

	if !reachable["a"] || !reachable["b"] {
		t.Error("cycle should still mark both as reachable")
	}
}
