package semantic

import (
	"testing"

	"github.com/foundry-zero/allium/internal/ast"
	"github.com/foundry-zero/allium/internal/report"
)

func warningSpec() *ast.Spec {
	return &ast.Spec{
		File: "test.allium.json",
		Entities: []ast.Entity{
			{
				Name: "Order",
				Fields: []ast.Field{
					{Name: "status", Type: ast.FieldType{Kind: "inline_enum", Values: []string{"pending", "shipped", "delivered"}}},
					{Name: "total", Type: ast.FieldType{Kind: "primitive", Value: "Integer"}},
					{Name: "created_at", Type: ast.FieldType{Kind: "primitive", Value: "Timestamp"}},
				},
				Relationships: []ast.Relationship{
					{Name: "customer", TargetEntity: "User", ForeignKey: "user_id", Cardinality: "one"},
				},
			},
			{
				Name: "User",
				Fields: []ast.Field{
					{Name: "name", Type: ast.FieldType{Kind: "primitive", Value: "String"}},
				},
			},
		},
		Actors: []ast.Actor{
			{Name: "Customer", IdentifiedBy: ast.IdentifiedBy{Entity: "User"}},
		},
		Surfaces: []ast.Surface{
			{
				Name:   "OrderView",
				Facing: ast.FacingClause{Binding: "viewer", Type: "Customer"},
				Context: &ast.ContextClause{
					Binding: "order",
					Type:    "Order",
				},
				Exposes: []ast.ExposesItem{
					{Expression: &ast.Expression{Kind: "field_access", Object: &ast.Expression{Kind: "field_access", Field: "order"}, Field: "status"}},
				},
				Provides: []ast.ProvidesItem{
					{
						Kind:    "action",
						Trigger: "submit_order",
						When:    &ast.Expression{Kind: "field_access", Object: &ast.Expression{Kind: "field_access", Field: "viewer"}, Field: "active"},
					},
				},
			},
		},
		Rules: []ast.Rule{
			{
				Name:    "SubmitOrder",
				Trigger: ast.Trigger{Kind: "external_stimulus", Name: "submit_order"},
				Ensures: []ast.EnsuresClause{
					{Kind: "state_change"},
				},
			},
		},
	}
}

// Helper to count findings by rule
func warnFindings(findings []report.Finding, rule string) []report.Finding {
	var result []report.Finding
	for _, f := range findings {
		if f.Rule == rule {
			result = append(result, f)
		}
	}
	return result
}

// ---- WARN-01 ----

func TestCheckWarnings_WARN01_ExternalNoSpec(t *testing.T) {
	spec := warningSpec()
	spec.ExternalEntities = []ast.ExternalEntity{
		{Name: "Payment", Fields: []ast.Field{{Name: "amount", Type: ast.FieldType{Kind: "primitive", Value: "Integer"}}}},
	}
	// No use declarations
	spec.UseDeclarations = nil
	st := BuildSymbolTable(spec)
	findings := CheckWarnings(spec, st)

	w01 := warnFindings(findings, "WARN-01")
	if len(w01) == 0 {
		t.Fatal("expected WARN-01 for external entity with no use declarations")
	}
}

func TestCheckWarnings_WARN01_HasUseDeclarations(t *testing.T) {
	spec := warningSpec()
	spec.ExternalEntities = []ast.ExternalEntity{
		{Name: "Payment"},
	}
	spec.UseDeclarations = []ast.UseDeclaration{
		{Coordinate: "org.example:payments", Alias: "payments"},
	}
	st := BuildSymbolTable(spec)
	findings := CheckWarnings(spec, st)

	w01 := warnFindings(findings, "WARN-01")
	if len(w01) > 0 {
		t.Error("should not fire WARN-01 when use declarations exist")
	}
}

// ---- WARN-02 ----

func TestCheckWarnings_WARN02_OpenQuestions(t *testing.T) {
	spec := warningSpec()
	spec.OpenQuestions = []string{"How should refunds work?"}
	st := BuildSymbolTable(spec)
	findings := CheckWarnings(spec, st)

	w02 := warnFindings(findings, "WARN-02")
	if len(w02) == 0 {
		t.Fatal("expected WARN-02 for open questions")
	}
}

func TestCheckWarnings_WARN02_NoOpenQuestions(t *testing.T) {
	spec := warningSpec()
	spec.OpenQuestions = nil
	st := BuildSymbolTable(spec)
	findings := CheckWarnings(spec, st)

	w02 := warnFindings(findings, "WARN-02")
	if len(w02) > 0 {
		t.Error("should not fire WARN-02 when no open questions")
	}
}

// ---- WARN-03 ----

func TestCheckWarnings_WARN03_DeferredNoHint(t *testing.T) {
	spec := warningSpec()
	spec.Deferred = []ast.Deferred{
		{Name: "PaymentProcessing", Method: "custom", LocationHint: nil},
	}
	st := BuildSymbolTable(spec)
	findings := CheckWarnings(spec, st)

	w03 := warnFindings(findings, "WARN-03")
	if len(w03) == 0 {
		t.Fatal("expected WARN-03 for deferred with nil location_hint")
	}
}

func TestCheckWarnings_WARN03_DeferredEmptyHint(t *testing.T) {
	spec := warningSpec()
	empty := ""
	spec.Deferred = []ast.Deferred{
		{Name: "PaymentProcessing", Method: "custom", LocationHint: &empty},
	}
	st := BuildSymbolTable(spec)
	findings := CheckWarnings(spec, st)

	w03 := warnFindings(findings, "WARN-03")
	if len(w03) == 0 {
		t.Fatal("expected WARN-03 for deferred with empty location_hint")
	}
}

func TestCheckWarnings_WARN03_DeferredWithHint(t *testing.T) {
	spec := warningSpec()
	hint := "https://example.com/spec"
	spec.Deferred = []ast.Deferred{
		{Name: "PaymentProcessing", Method: "custom", LocationHint: &hint},
	}
	st := BuildSymbolTable(spec)
	findings := CheckWarnings(spec, st)

	w03 := warnFindings(findings, "WARN-03")
	if len(w03) > 0 {
		t.Error("should not fire WARN-03 when location_hint is provided")
	}
}

// ---- WARN-04 ----

func TestCheckWarnings_WARN04_UnusedEntity(t *testing.T) {
	spec := warningSpec()
	// Add an entity not referenced anywhere
	spec.Entities = append(spec.Entities, ast.Entity{
		Name:   "Orphan",
		Fields: []ast.Field{{Name: "x", Type: ast.FieldType{Kind: "primitive", Value: "String"}}},
	})
	st := BuildSymbolTable(spec)
	findings := CheckWarnings(spec, st)

	w04 := warnFindings(findings, "WARN-04")
	found := false
	for _, f := range w04 {
		if f.Message == "Unused entity 'Orphan'" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected WARN-04 for unreferenced entity 'Orphan'")
	}
}

func TestCheckWarnings_WARN04_AllUsed(t *testing.T) {
	spec := warningSpec()
	st := BuildSymbolTable(spec)
	findings := CheckWarnings(spec, st)

	w04 := warnFindings(findings, "WARN-04")
	// Order is referenced by surfaces/rules, User by relationships/actors
	for _, f := range w04 {
		t.Errorf("unexpected WARN-04: %s", f.Message)
	}
}

// ---- WARN-05 ----

func TestCheckWarnings_WARN05_ContradictoryRequires(t *testing.T) {
	spec := warningSpec()
	spec.Rules[0].Requires = []ast.Expression{
		{
			Kind:     "comparison",
			Operator: "=",
			Left:     &ast.Expression{Kind: "field_access", Field: "status"},
			Right:    &ast.Expression{Kind: "literal", Type: "string", LitValue: []byte(`"pending"`)},
		},
		{
			Kind:     "comparison",
			Operator: "=",
			Left:     &ast.Expression{Kind: "field_access", Field: "status"},
			Right:    &ast.Expression{Kind: "literal", Type: "string", LitValue: []byte(`"shipped"`)},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckWarnings(spec, st)

	w05 := warnFindings(findings, "WARN-05")
	if len(w05) == 0 {
		t.Fatal("expected WARN-05 for contradictory requires")
	}
}

func TestCheckWarnings_WARN05_ConsistentRequires(t *testing.T) {
	spec := warningSpec()
	spec.Rules[0].Requires = []ast.Expression{
		{
			Kind:     "comparison",
			Operator: "=",
			Left:     &ast.Expression{Kind: "field_access", Field: "status"},
			Right:    &ast.Expression{Kind: "literal", Type: "string", LitValue: []byte(`"pending"`)},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckWarnings(spec, st)

	w05 := warnFindings(findings, "WARN-05")
	if len(w05) > 0 {
		t.Error("should not fire WARN-05 for consistent requires")
	}
}

// ---- WARN-06 ----

func TestCheckWarnings_WARN06_TemporalNoGuard(t *testing.T) {
	spec := warningSpec()
	spec.Rules = append(spec.Rules, ast.Rule{
		Name:    "ExpireOrder",
		Trigger: ast.Trigger{Kind: "temporal", Entity: "Order", Field: "created_at"},
		// No requires
		Ensures: []ast.EnsuresClause{{Kind: "state_change"}},
	})
	st := BuildSymbolTable(spec)
	findings := CheckWarnings(spec, st)

	w06 := warnFindings(findings, "WARN-06")
	if len(w06) == 0 {
		t.Fatal("expected WARN-06 for temporal rule without guard")
	}
}

func TestCheckWarnings_WARN06_TemporalWithGuard(t *testing.T) {
	spec := warningSpec()
	spec.Rules = append(spec.Rules, ast.Rule{
		Name:    "ExpireOrder",
		Trigger: ast.Trigger{Kind: "temporal", Entity: "Order", Field: "created_at"},
		Requires: []ast.Expression{
			{Kind: "comparison", Operator: "=",
				Left:  &ast.Expression{Kind: "field_access", Field: "status"},
				Right: &ast.Expression{Kind: "literal", Type: "string", LitValue: []byte(`"pending"`)},
			},
		},
		Ensures: []ast.EnsuresClause{{Kind: "state_change"}},
	})
	st := BuildSymbolTable(spec)
	findings := CheckWarnings(spec, st)

	w06 := warnFindings(findings, "WARN-06")
	if len(w06) > 0 {
		t.Error("should not fire WARN-06 for temporal rule with guard")
	}
}

// ---- WARN-09 ----

func TestCheckWarnings_WARN09_UnusedActor(t *testing.T) {
	spec := warningSpec()
	spec.Actors = append(spec.Actors, ast.Actor{
		Name:         "Admin",
		IdentifiedBy: ast.IdentifiedBy{Entity: "User"},
	})
	st := BuildSymbolTable(spec)
	findings := CheckWarnings(spec, st)

	w09 := warnFindings(findings, "WARN-09")
	found := false
	for _, f := range w09 {
		if f.Message == "Unused actor 'Admin'" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected WARN-09 for actor not referenced in any surface facing")
	}
}

func TestCheckWarnings_WARN09_ActorUsedInFacing(t *testing.T) {
	spec := warningSpec()
	// Customer is used in OrderView's facing
	st := BuildSymbolTable(spec)
	findings := CheckWarnings(spec, st)

	w09 := warnFindings(findings, "WARN-09")
	for _, f := range w09 {
		if f.Message == "Unused actor 'Customer'" {
			t.Error("Customer is used in surface facing, should not trigger WARN-09")
		}
	}
}

// ---- WARN-12 ----

func TestCheckWarnings_WARN12_OverlappingRequires(t *testing.T) {
	spec := warningSpec()
	// Two rules sharing the same trigger with no requires
	spec.Rules = []ast.Rule{
		{
			Name:    "RuleA",
			Trigger: ast.Trigger{Kind: "external_stimulus", Name: "do_thing"},
			Ensures: []ast.EnsuresClause{{Kind: "state_change"}},
		},
		{
			Name:    "RuleB",
			Trigger: ast.Trigger{Kind: "external_stimulus", Name: "do_thing"},
			Ensures: []ast.EnsuresClause{{Kind: "state_change"}},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckWarnings(spec, st)

	w12 := warnFindings(findings, "WARN-12")
	if len(w12) == 0 {
		t.Fatal("expected WARN-12 for overlapping preconditions")
	}
}

func TestCheckWarnings_WARN12_DisjointTriggers(t *testing.T) {
	spec := warningSpec()
	st := BuildSymbolTable(spec)
	findings := CheckWarnings(spec, st)

	w12 := warnFindings(findings, "WARN-12")
	if len(w12) > 0 {
		t.Errorf("should not fire WARN-12 when no overlapping, got %d", len(w12))
	}
}

// ---- WARN-14 ----

func TestCheckWarnings_WARN14_TrivialActor(t *testing.T) {
	spec := warningSpec()
	spec.Actors[0].IdentifiedBy.Condition = &ast.Expression{Kind: "literal", Type: "boolean", LitValue: []byte("true")}
	st := BuildSymbolTable(spec)
	findings := CheckWarnings(spec, st)

	w14 := warnFindings(findings, "WARN-14")
	if len(w14) == 0 {
		t.Fatal("expected WARN-14 for trivial boolean literal condition")
	}
}

func TestCheckWarnings_WARN14_NilCondition(t *testing.T) {
	spec := warningSpec()
	spec.Actors[0].IdentifiedBy.Condition = nil
	st := BuildSymbolTable(spec)
	findings := CheckWarnings(spec, st)

	w14 := warnFindings(findings, "WARN-14")
	if len(w14) > 0 {
		t.Error("should not fire WARN-14 when condition is nil")
	}
}

// ---- WARN-15 ----

func TestCheckWarnings_WARN15_AllConditionalWithEmptyElse(t *testing.T) {
	spec := warningSpec()
	spec.Rules[0].Ensures = []ast.EnsuresClause{
		{
			Kind:      "conditional",
			Condition: &ast.Expression{Kind: "literal", Type: "boolean"},
			Then:      []ast.EnsuresClause{{Kind: "state_change"}},
			Else:      nil, // empty else path
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckWarnings(spec, st)

	w15 := warnFindings(findings, "WARN-15")
	if len(w15) == 0 {
		t.Fatal("expected WARN-15 for all-conditional ensures with empty else")
	}
}

func TestCheckWarnings_WARN15_NonConditionalPresent(t *testing.T) {
	spec := warningSpec()
	spec.Rules[0].Ensures = []ast.EnsuresClause{
		{Kind: "state_change"},
		{
			Kind:      "conditional",
			Condition: &ast.Expression{Kind: "literal", Type: "boolean"},
			Then:      []ast.EnsuresClause{{Kind: "state_change"}},
			Else:      nil,
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckWarnings(spec, st)

	w15 := warnFindings(findings, "WARN-15")
	if len(w15) > 0 {
		t.Error("should not fire WARN-15 when non-conditional ensures present")
	}
}

// ---- WARN-16 ----

func TestCheckWarnings_WARN16_OptionalTemporal(t *testing.T) {
	spec := warningSpec()
	// Make created_at optional
	spec.Entities[0].Fields[2] = ast.Field{
		Name: "created_at",
		Type: ast.FieldType{Kind: "optional", Inner: &ast.FieldType{Kind: "primitive", Value: "Timestamp"}},
	}
	spec.Rules = append(spec.Rules, ast.Rule{
		Name:    "ExpireOrder",
		Trigger: ast.Trigger{Kind: "temporal", Entity: "Order", Field: "created_at"},
		Requires: []ast.Expression{
			{Kind: "literal", Type: "boolean"},
		},
		Ensures: []ast.EnsuresClause{{Kind: "state_change"}},
	})
	st := BuildSymbolTable(spec)
	findings := CheckWarnings(spec, st)

	w16 := warnFindings(findings, "WARN-16")
	if len(w16) == 0 {
		t.Fatal("expected WARN-16 for temporal trigger on optional field")
	}
}

func TestCheckWarnings_WARN16_RequiredTemporal(t *testing.T) {
	spec := warningSpec()
	spec.Rules = append(spec.Rules, ast.Rule{
		Name:    "ExpireOrder",
		Trigger: ast.Trigger{Kind: "temporal", Entity: "Order", Field: "created_at"},
		Requires: []ast.Expression{
			{Kind: "literal", Type: "boolean"},
		},
		Ensures: []ast.EnsuresClause{{Kind: "state_change"}},
	})
	st := BuildSymbolTable(spec)
	findings := CheckWarnings(spec, st)

	w16 := warnFindings(findings, "WARN-16")
	if len(w16) > 0 {
		t.Error("should not fire WARN-16 when field is required (not optional)")
	}
}

func TestCheckWarnings_WARN16_ConditionOptionalField(t *testing.T) {
	spec := warningSpec()
	// Add an optional Timestamp field
	spec.Entities[0].Fields = append(spec.Entities[0].Fields, ast.Field{
		Name: "expires_at",
		Type: ast.FieldType{Kind: "optional", Inner: &ast.FieldType{Kind: "primitive", Value: "Timestamp"}},
	})
	// Temporal trigger with condition expression referencing the optional field
	// (no explicit Field property — this is how real temporal triggers work)
	spec.Rules = append(spec.Rules, ast.Rule{
		Name: "ExpireOrder",
		Trigger: ast.Trigger{
			Kind:    "temporal",
			Entity:  "Order",
			Binding: "order",
			Condition: &ast.Expression{
				Kind:     "comparison",
				Operator: "<",
				Left: &ast.Expression{
					Kind: "field_access",
					Object: &ast.Expression{
						Kind:  "field_access",
						Field: "order",
					},
					Field: "expires_at",
				},
				Right: &ast.Expression{
					Kind:     "function_call",
					FuncName: "now",
				},
			},
		},
		Requires: []ast.Expression{
			{Kind: "literal", Type: "boolean"},
		},
		Ensures: []ast.EnsuresClause{{Kind: "state_change"}},
	})
	st := BuildSymbolTable(spec)
	findings := CheckWarnings(spec, st)

	w16 := warnFindings(findings, "WARN-16")
	if len(w16) == 0 {
		t.Fatal("expected WARN-16 for temporal trigger condition on optional field")
	}
}

func TestCheckWarnings_WARN16_ConditionRequiredField(t *testing.T) {
	spec := warningSpec()
	// created_at is already a required Timestamp field on Order
	spec.Rules = append(spec.Rules, ast.Rule{
		Name: "ExpireOrder",
		Trigger: ast.Trigger{
			Kind:    "temporal",
			Entity:  "Order",
			Binding: "order",
			Condition: &ast.Expression{
				Kind:     "comparison",
				Operator: "<",
				Left: &ast.Expression{
					Kind: "field_access",
					Object: &ast.Expression{
						Kind:  "field_access",
						Field: "order",
					},
					Field: "created_at",
				},
				Right: &ast.Expression{
					Kind:     "function_call",
					FuncName: "now",
				},
			},
		},
		Requires: []ast.Expression{
			{Kind: "literal", Type: "boolean"},
		},
		Ensures: []ast.EnsuresClause{{Kind: "state_change"}},
	})
	st := BuildSymbolTable(spec)
	findings := CheckWarnings(spec, st)

	w16 := warnFindings(findings, "WARN-16")
	if len(w16) > 0 {
		t.Error("should not fire WARN-16 when field referenced in condition is required")
	}
}

// ---- WARN-17 ----

func TestCheckWarnings_WARN17_RawEntityWithActors(t *testing.T) {
	spec := warningSpec()
	// Surface facing raw entity "User" instead of actor "Customer"
	spec.Surfaces = append(spec.Surfaces, ast.Surface{
		Name:   "UserView",
		Facing: ast.FacingClause{Binding: "u", Type: "User"},
	})
	st := BuildSymbolTable(spec)
	findings := CheckWarnings(spec, st)

	w17 := warnFindings(findings, "WARN-17")
	if len(w17) == 0 {
		t.Fatal("expected WARN-17 for raw entity type used when actors available")
	}
}

func TestCheckWarnings_WARN17_ActorUsed(t *testing.T) {
	spec := warningSpec()
	// OrderView faces "Customer" (an actor) — no warning
	st := BuildSymbolTable(spec)
	findings := CheckWarnings(spec, st)

	w17 := warnFindings(findings, "WARN-17")
	if len(w17) > 0 {
		t.Errorf("should not fire WARN-17 when facing uses actor type, got %d", len(w17))
	}
}

// ---- WARN-18 ----

func TestCheckWarnings_WARN18_TransitionsOnCreation(t *testing.T) {
	spec := warningSpec()
	// A rule creates Order with status=pending
	spec.Rules = append(spec.Rules, ast.Rule{
		Name:    "CreateOrder",
		Trigger: ast.Trigger{Kind: "external_stimulus", Name: "create_order"},
		Ensures: []ast.EnsuresClause{
			{
				Kind:   "entity_creation",
				Entity: "Order",
				Fields: map[string]ast.Expression{
					"status": {Kind: "literal", Type: "string", LitValue: []byte(`"pending"`)},
				},
			},
		},
	})
	// Another rule transitions_to "pending" — fires on creation
	spec.Rules = append(spec.Rules, ast.Rule{
		Name:    "ResetOrder",
		Trigger: ast.Trigger{Kind: "state_transition", Entity: "Order", Field: "status", ToValue: "pending"},
		Ensures: []ast.EnsuresClause{{Kind: "state_change"}},
	})
	st := BuildSymbolTable(spec)
	findings := CheckWarnings(spec, st)

	w18 := warnFindings(findings, "WARN-18")
	if len(w18) == 0 {
		t.Fatal("expected WARN-18 for transitions_to on a creation value")
	}
}

func TestCheckWarnings_WARN18_TransitionsToNonCreation(t *testing.T) {
	spec := warningSpec()
	spec.Rules = append(spec.Rules, ast.Rule{
		Name:    "ShipOrder",
		Trigger: ast.Trigger{Kind: "state_transition", Entity: "Order", Field: "status", ToValue: "shipped"},
		Ensures: []ast.EnsuresClause{{Kind: "state_change"}},
	})
	st := BuildSymbolTable(spec)
	findings := CheckWarnings(spec, st)

	w18 := warnFindings(findings, "WARN-18")
	if len(w18) > 0 {
		t.Error("should not fire WARN-18 when to_value is not a creation value")
	}
}

// ---- WARN-19 ----

func TestCheckWarnings_WARN19_DuplicateInlineEnums(t *testing.T) {
	spec := warningSpec()
	// Add a field with same inline enum values as "status"
	spec.Entities[0].Fields = append(spec.Entities[0].Fields, ast.Field{
		Name: "previous_status",
		Type: ast.FieldType{Kind: "inline_enum", Values: []string{"pending", "shipped", "delivered"}},
	})
	st := BuildSymbolTable(spec)
	findings := CheckWarnings(spec, st)

	w19 := warnFindings(findings, "WARN-19")
	if len(w19) == 0 {
		t.Fatal("expected WARN-19 for duplicate inline enum sets")
	}
}

func TestCheckWarnings_WARN19_UniqueInlineEnums(t *testing.T) {
	spec := warningSpec()
	spec.Entities[0].Fields = append(spec.Entities[0].Fields, ast.Field{
		Name: "priority",
		Type: ast.FieldType{Kind: "inline_enum", Values: []string{"low", "medium", "high"}},
	})
	st := BuildSymbolTable(spec)
	findings := CheckWarnings(spec, st)

	w19 := warnFindings(findings, "WARN-19")
	if len(w19) > 0 {
		t.Error("should not fire WARN-19 for unique inline enum sets")
	}
}

// ---- Clean spec: no warnings on baseline ----

func TestCheckWarnings_Clean(t *testing.T) {
	spec := warningSpec()
	st := BuildSymbolTable(spec)
	findings := CheckWarnings(spec, st)

	// The baseline spec might trigger WARN-04 for entities not referenced by rules
	// but Order is referenced by surfaces and User by relationships/actors
	for _, f := range findings {
		t.Errorf("unexpected warning: [%s] %s at %s", f.Rule, f.Message, f.Location.Path)
	}
}

// ---- Empty spec: no warnings ----

func TestCheckWarnings_EmptySpec(t *testing.T) {
	spec := &ast.Spec{File: "test.allium.json"}
	st := BuildSymbolTable(spec)
	findings := CheckWarnings(spec, st)

	if len(findings) > 0 {
		for _, f := range findings {
			t.Errorf("unexpected: [%s] %s", f.Rule, f.Message)
		}
	}
}

// ---- All findings are warnings ----

func TestCheckWarnings_AllFindingsAreWarnings(t *testing.T) {
	spec := warningSpec()
	spec.OpenQuestions = []string{"Something?"}
	spec.ExternalEntities = []ast.ExternalEntity{{Name: "Ext"}}
	spec.Deferred = []ast.Deferred{{Name: "D", Method: "m", LocationHint: nil}}
	spec.Actors = append(spec.Actors, ast.Actor{Name: "Admin", IdentifiedBy: ast.IdentifiedBy{Entity: "User"}})
	st := BuildSymbolTable(spec)
	findings := CheckWarnings(spec, st)

	for _, f := range findings {
		if f.Severity != report.SeverityWarning {
			t.Errorf("expected warning severity, got %s for %s", f.Severity, f.Rule)
		}
	}
}
