package semantic

import (
	"testing"

	"github.com/foundry-zero/allium/internal/ast"
)

func sumTypeSpec() *ast.Spec {
	return &ast.Spec{
		File: "test.allium.json",
		Entities: []ast.Entity{
			{
				Name: "Node",
				Fields: []ast.Field{
					{Name: "path", Type: ast.FieldType{Kind: "primitive", Value: "String"}},
					{Name: "kind", Type: ast.FieldType{Kind: "inline_enum", Values: []string{"Branch", "Leaf"}}},
				},
			},
		},
		Variants: []ast.Variant{
			{Name: "Branch", BaseEntity: "Node", Fields: []ast.Field{
				{Name: "children", Type: ast.FieldType{Kind: "list", Element: &ast.FieldType{Kind: "entity_ref", Entity: "Node"}}},
			}},
			{Name: "Leaf", BaseEntity: "Node"},
		},
		Rules: []ast.Rule{
			{
				Name:    "CreateBranch",
				Trigger: ast.Trigger{Kind: "external_stimulus", Name: "create_branch"},
				Ensures: []ast.EnsuresClause{
					{Kind: "entity_creation", Entity: "Branch"},
				},
			},
		},
	}
}

func TestCheckSumTypes_Clean(t *testing.T) {
	spec := sumTypeSpec()
	st := BuildSymbolTable(spec)
	findings := CheckSumTypes(spec, st)

	if len(findings) > 0 {
		for _, f := range findings {
			t.Errorf("unexpected: [%s] %s", f.Rule, f.Message)
		}
	}
}

func TestCheckSumTypes_RULE16_MissingVariantDecl(t *testing.T) {
	spec := sumTypeSpec()
	// Remove the Leaf variant declaration
	spec.Variants = spec.Variants[:1] // only Branch
	st := BuildSymbolTable(spec)
	findings := CheckSumTypes(spec, st)

	r16 := findingsWithRule(findings, "RULE-16")
	if len(r16) == 0 {
		t.Fatal("expected RULE-16 for missing Leaf variant declaration")
	}
}

func TestCheckSumTypes_RULE16_WrongBaseEntity(t *testing.T) {
	spec := sumTypeSpec()
	// Point Leaf at wrong base entity
	spec.Variants[1].BaseEntity = "OtherEntity"
	st := BuildSymbolTable(spec)
	findings := CheckSumTypes(spec, st)

	r16 := findingsWithRule(findings, "RULE-16")
	if len(r16) == 0 {
		t.Fatal("expected RULE-16 for variant with wrong base entity")
	}
}

func TestCheckSumTypes_RULE17_UnlistedVariant(t *testing.T) {
	spec := sumTypeSpec()
	// Add a variant not listed in discriminator
	spec.Variants = append(spec.Variants, ast.Variant{Name: "Stem", BaseEntity: "Node"})
	st := BuildSymbolTable(spec)
	findings := CheckSumTypes(spec, st)

	r17 := findingsWithRule(findings, "RULE-17")
	if len(r17) == 0 {
		t.Fatal("expected RULE-17 for unlisted variant Stem")
	}
}

func TestCheckSumTypes_RULE17_NoDiscriminator(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Entities: []ast.Entity{
			{Name: "Simple", Fields: []ast.Field{
				{Name: "name", Type: ast.FieldType{Kind: "primitive", Value: "String"}},
			}},
		},
		Variants: []ast.Variant{
			{Name: "ExtendedSimple", BaseEntity: "Simple"},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckSumTypes(spec, st)

	r17 := findingsWithRule(findings, "RULE-17")
	if len(r17) == 0 {
		t.Fatal("expected RULE-17 for variant with no discriminator on base entity")
	}
}

func TestCheckSumTypes_RULE19_BaseEntityCreation(t *testing.T) {
	spec := sumTypeSpec()
	// Change creation to use base entity name "Node" instead of "Branch"
	spec.Rules[0].Ensures[0].Entity = "Node"
	st := BuildSymbolTable(spec)
	findings := CheckSumTypes(spec, st)

	r19 := findingsWithRule(findings, "RULE-19")
	if len(r19) == 0 {
		t.Fatal("expected RULE-19 for base entity creation with discriminator")
	}
}

func TestCheckSumTypes_RULE19_VariantCreation_OK(t *testing.T) {
	spec := sumTypeSpec()
	// Creating "Branch" (a variant) — should be fine
	st := BuildSymbolTable(spec)
	findings := CheckSumTypes(spec, st)

	r19 := findingsWithRule(findings, "RULE-19")
	if len(r19) > 0 {
		t.Error("variant name creation should not trigger RULE-19")
	}
}

func TestCheckSumTypes_RULE19_NestedConditional(t *testing.T) {
	spec := sumTypeSpec()
	spec.Rules[0].Ensures = []ast.EnsuresClause{
		{
			Kind: "conditional",
			Then: []ast.EnsuresClause{
				{Kind: "entity_creation", Entity: "Node"}, // base entity in conditional
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckSumTypes(spec, st)

	r19 := findingsWithRule(findings, "RULE-19")
	if len(r19) == 0 {
		t.Fatal("expected RULE-19 for base entity creation in conditional")
	}
}

func TestCheckSumTypes_NoSumTypes(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Entities: []ast.Entity{
			{Name: "Simple", Fields: []ast.Field{
				{Name: "status", Type: ast.FieldType{Kind: "inline_enum", Values: []string{"active", "inactive"}}},
			}},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckSumTypes(spec, st)

	if len(findings) > 0 {
		t.Errorf("no sum types should produce no findings, got %d", len(findings))
	}
}

func TestIsDiscriminator(t *testing.T) {
	tests := []struct {
		values []string
		want   bool
	}{
		{[]string{"Branch", "Leaf"}, true},
		{[]string{"active", "inactive"}, false},      // lowercase = enum
		{[]string{"Active", "inactive"}, false},       // mixed
		{[]string{}, false},                           // empty
		{[]string{"A"}, true},                         // single PascalCase
		{[]string{"PremiumUser", "BasicUser"}, true},
	}
	for _, tt := range tests {
		got := isDiscriminator(tt.values)
		if got != tt.want {
			t.Errorf("isDiscriminator(%v) = %v, want %v", tt.values, got, tt.want)
		}
	}
}
