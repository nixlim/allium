package semantic

import (
	"testing"

	"github.com/foundry-zero/allium/internal/ast"
)

func surfaceSpec() *ast.Spec {
	return &ast.Spec{
		File: "test.allium.json",
		Entities: []ast.Entity{
			{
				Name: "Order",
				Fields: []ast.Field{
					{Name: "status", Type: ast.FieldType{Kind: "primitive", Value: "String"}},
					{Name: "items", Type: ast.FieldType{Kind: "list", Element: &ast.FieldType{Kind: "primitive", Value: "String"}}},
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
			{Name: "SubmitOrder", Trigger: ast.Trigger{Kind: "external_stimulus", Name: "submit_order"}},
		},
	}
}

func TestCheckSurfaces_Clean(t *testing.T) {
	spec := surfaceSpec()
	st := BuildSymbolTable(spec)
	findings := CheckSurfaces(spec, st)

	if len(findings) > 0 {
		for _, f := range findings {
			t.Errorf("unexpected: [%s] %s at %s", f.Rule, f.Message, f.Location.Path)
		}
	}
}

func TestCheckSurfaces_RULE29_UnreachableExposes(t *testing.T) {
	spec := surfaceSpec()
	// Add an exposes item referencing an unknown root binding
	spec.Surfaces[0].Exposes = append(spec.Surfaces[0].Exposes,
		ast.ExposesItem{
			Expression: &ast.Expression{Kind: "field_access", Object: &ast.Expression{Kind: "field_access", Field: "product"}, Field: "name"},
		},
	)
	st := BuildSymbolTable(spec)
	findings := CheckSurfaces(spec, st)

	r29 := findingsWithRule(findings, "RULE-29")
	if len(r29) == 0 {
		t.Fatal("expected RULE-29 for unreachable exposes path")
	}
}

func TestCheckSurfaces_RULE29_ReachableViaLetBinding(t *testing.T) {
	spec := surfaceSpec()
	// Add a let binding and an exposes that uses it
	spec.Surfaces[0].LetBindings = []ast.LetBinding{
		{Name: "total", Expression: &ast.Expression{Kind: "literal", Type: "integer"}},
	}
	spec.Surfaces[0].Exposes = append(spec.Surfaces[0].Exposes,
		ast.ExposesItem{
			Expression: &ast.Expression{Kind: "field_access", Field: "total"},
		},
	)
	st := BuildSymbolTable(spec)
	findings := CheckSurfaces(spec, st)

	r29 := findingsWithRule(findings, "RULE-29")
	if len(r29) > 0 {
		t.Error("let binding path should be reachable")
	}
}

func TestCheckSurfaces_RULE32_UnusedFacing(t *testing.T) {
	spec := surfaceSpec()
	// Remove all references to "viewer"
	spec.Surfaces[0].Provides[0].When = nil
	st := BuildSymbolTable(spec)
	findings := CheckSurfaces(spec, st)

	r32 := findingsWithRule(findings, "RULE-32")
	if len(r32) == 0 {
		t.Fatal("expected RULE-32 for unused facing binding 'viewer'")
	}
}

func TestCheckSurfaces_RULE32_UnusedContext(t *testing.T) {
	spec := surfaceSpec()
	// Remove all references to "order" from exposes
	spec.Surfaces[0].Exposes = nil
	st := BuildSymbolTable(spec)
	findings := CheckSurfaces(spec, st)

	r32 := findingsWithRule(findings, "RULE-32")
	found := false
	for _, f := range r32 {
		if f.Location.Path == "$.surfaces[0].context.binding" {
			found = true
		}
	}
	if !found {
		t.Error("expected RULE-32 for unused context binding 'order'")
	}
}

func TestCheckSurfaces_RULE32_BothUsed(t *testing.T) {
	spec := surfaceSpec()
	st := BuildSymbolTable(spec)
	findings := CheckSurfaces(spec, st)

	r32 := findingsWithRule(findings, "RULE-32")
	if len(r32) > 0 {
		t.Errorf("both bindings used, should not trigger RULE-32, got %d", len(r32))
	}
}

func TestCheckSurfaces_RULE32_NoContext(t *testing.T) {
	spec := surfaceSpec()
	spec.Surfaces[0].Context = nil
	// Remove exposes that reference order
	spec.Surfaces[0].Exposes = nil
	// Keep provides that reference viewer
	st := BuildSymbolTable(spec)
	findings := CheckSurfaces(spec, st)

	r32 := findingsWithRule(findings, "RULE-32")
	// Should only check context if it exists
	for _, f := range r32 {
		if f.Location.Path == "$.surfaces[0].context.binding" {
			t.Error("should not check context binding when context is nil")
		}
	}
}

func TestCheckSurfaces_RULE32_UsedInRelated(t *testing.T) {
	spec := surfaceSpec()
	spec.Surfaces[0].Provides[0].When = nil // remove viewer from provides
	// Add a related item that uses viewer
	spec.Surfaces = append(spec.Surfaces, ast.Surface{
		Name:   "OtherSurface",
		Facing: ast.FacingClause{Binding: "u", Type: "Customer"},
	})
	spec.Surfaces[0].Related = []ast.RelatedItem{
		{
			Surface:           "OtherSurface",
			ContextExpression: &ast.Expression{Kind: "field_access", Field: "viewer"},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckSurfaces(spec, st)

	r32 := findingsWithRule(findings, "RULE-32")
	for _, f := range r32 {
		if f.Location.Path == "$.surfaces[0].facing.binding" {
			t.Error("viewer is used in related, should not trigger RULE-32")
		}
	}
}

func TestCheckSurfaces_RULE33_UnreachableWhenInProvides(t *testing.T) {
	spec := surfaceSpec()
	// Add a provides action with a when condition referencing an unknown root
	spec.Surfaces[0].Provides = append(spec.Surfaces[0].Provides, ast.ProvidesItem{
		Kind:    "action",
		Trigger: "submit_order",
		When: &ast.Expression{
			Kind: "field_access",
			Object: &ast.Expression{Kind: "field_access", Field: "unknown"},
			Field: "active",
		},
	})
	st := BuildSymbolTable(spec)
	findings := CheckSurfaces(spec, st)

	r33 := findingsWithRule(findings, "RULE-33")
	if len(r33) == 0 {
		t.Fatal("expected RULE-33 for unreachable when condition in provides")
	}
}

func TestCheckSurfaces_RULE33_UnreachableWhenInExposes(t *testing.T) {
	spec := surfaceSpec()
	// Add an exposes item with when condition referencing unknown root
	spec.Surfaces[0].Exposes = append(spec.Surfaces[0].Exposes, ast.ExposesItem{
		Expression: &ast.Expression{Kind: "field_access", Object: &ast.Expression{Kind: "field_access", Field: "order"}, Field: "status"},
		When: &ast.Expression{
			Kind: "field_access",
			Object: &ast.Expression{Kind: "field_access", Field: "unknown_binding"},
			Field: "visible",
		},
	})
	st := BuildSymbolTable(spec)
	findings := CheckSurfaces(spec, st)

	r33 := findingsWithRule(findings, "RULE-33")
	if len(r33) == 0 {
		t.Fatal("expected RULE-33 for unreachable when condition in exposes")
	}
}

func TestCheckSurfaces_RULE33_ReachableWhen(t *testing.T) {
	spec := surfaceSpec()
	// When condition referencing valid bindings should not fire
	st := BuildSymbolTable(spec)
	findings := CheckSurfaces(spec, st)

	r33 := findingsWithRule(findings, "RULE-33")
	if len(r33) > 0 {
		t.Errorf("reachable when conditions should not trigger RULE-33, got %d", len(r33))
	}
}

func TestCheckSurfaces_RULE33_ForEachNestedWhen(t *testing.T) {
	spec := surfaceSpec()
	// for_each introduces binding "item", nested action when referencing "item" is OK
	spec.Surfaces[0].Provides = []ast.ProvidesItem{
		{
			Kind:    "for_each",
			Binding: "item",
			Collection: &ast.Expression{
				Kind:   "field_access",
				Object: &ast.Expression{Kind: "field_access", Field: "order"},
				Field:  "items",
			},
			Items: []ast.ProvidesItem{
				{
					Kind:    "action",
					Trigger: "submit_order",
					When: &ast.Expression{
						Kind:   "field_access",
						Object: &ast.Expression{Kind: "field_access", Field: "item"},
						Field:  "active",
					},
				},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckSurfaces(spec, st)

	r33 := findingsWithRule(findings, "RULE-33")
	if len(r33) > 0 {
		t.Errorf("for_each binding should be reachable in nested items, got %d RULE-33", len(r33))
	}
}

func TestCheckSurfaces_RULE34_IterateOverString(t *testing.T) {
	spec := surfaceSpec()
	// for_each over order.status which is a String (not a collection)
	spec.Surfaces[0].Provides = []ast.ProvidesItem{
		{
			Kind:    "for_each",
			Binding: "s",
			Collection: &ast.Expression{
				Kind:   "field_access",
				Object: &ast.Expression{Kind: "field_access", Field: "order"},
				Field:  "status",
			},
			Items: []ast.ProvidesItem{
				{Kind: "action", Trigger: "submit_order"},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckSurfaces(spec, st)

	r34 := findingsWithRule(findings, "RULE-34")
	if len(r34) == 0 {
		t.Fatal("expected RULE-34 for iterating over String field")
	}
}

func TestCheckSurfaces_RULE34_IterateOverList(t *testing.T) {
	spec := surfaceSpec()
	// for_each over order.items which is a list (valid)
	spec.Surfaces[0].Provides = []ast.ProvidesItem{
		{
			Kind:    "for_each",
			Binding: "item",
			Collection: &ast.Expression{
				Kind:   "field_access",
				Object: &ast.Expression{Kind: "field_access", Field: "order"},
				Field:  "items",
			},
			Items: []ast.ProvidesItem{
				{Kind: "action", Trigger: "submit_order"},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckSurfaces(spec, st)

	r34 := findingsWithRule(findings, "RULE-34")
	if len(r34) > 0 {
		t.Errorf("iterating over list should not trigger RULE-34, got %d", len(r34))
	}
}

func TestCheckSurfaces_RULE34_IterateOverManyRelationship(t *testing.T) {
	spec := surfaceSpec()
	// Add a many-cardinality relationship
	spec.Entities[0].Relationships = []ast.Relationship{
		{Name: "line_items", TargetEntity: "LineItem", ForeignKey: "order_id", Cardinality: "many"},
	}
	spec.Surfaces[0].Provides = []ast.ProvidesItem{
		{
			Kind:    "for_each",
			Binding: "li",
			Collection: &ast.Expression{
				Kind:   "field_access",
				Object: &ast.Expression{Kind: "field_access", Field: "order"},
				Field:  "line_items",
			},
			Items: []ast.ProvidesItem{
				{Kind: "action", Trigger: "submit_order"},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckSurfaces(spec, st)

	r34 := findingsWithRule(findings, "RULE-34")
	if len(r34) > 0 {
		t.Errorf("iterating over many-cardinality relationship should not trigger RULE-34, got %d", len(r34))
	}
}

func TestCheckSurfaces_NoSurfaces(t *testing.T) {
	spec := &ast.Spec{File: "test.allium.json"}
	st := BuildSymbolTable(spec)
	findings := CheckSurfaces(spec, st)

	if len(findings) > 0 {
		t.Errorf("no surfaces should produce no findings, got %d", len(findings))
	}
}
