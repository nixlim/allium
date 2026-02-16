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

// sumTypeSpecSnakeCase returns a spec with snake_case inline_enum values,
// matching what real schema-validated specs would contain.
func sumTypeSpecSnakeCase() *ast.Spec {
	return &ast.Spec{
		File: "test.allium.json",
		Entities: []ast.Entity{
			{
				Name: "Payment",
				Fields: []ast.Field{
					{Name: "amount", Type: ast.FieldType{Kind: "primitive", Value: "Integer"}},
					{Name: "kind", Type: ast.FieldType{Kind: "inline_enum", Values: []string{"card_payment", "bank_transfer"}}},
				},
			},
		},
		Variants: []ast.Variant{
			{Name: "CardPayment", BaseEntity: "Payment", Fields: []ast.Field{
				{Name: "card_number", Type: ast.FieldType{Kind: "primitive", Value: "String"}},
			}},
			{Name: "BankTransfer", BaseEntity: "Payment", Fields: []ast.Field{
				{Name: "routing_number", Type: ast.FieldType{Kind: "primitive", Value: "String"}},
			}},
		},
		Rules: []ast.Rule{
			{
				Name:    "CreateCardPayment",
				Trigger: ast.Trigger{Kind: "external_stimulus", Name: "create_card_payment"},
				Ensures: []ast.EnsuresClause{
					{Kind: "entity_creation", Entity: "CardPayment"},
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

func TestCheckSumTypes_CleanSnakeCase(t *testing.T) {
	spec := sumTypeSpecSnakeCase()
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

func TestCheckSumTypes_RULE16_MissingVariantDeclSnakeCase(t *testing.T) {
	spec := sumTypeSpecSnakeCase()
	// Remove BankTransfer variant — "bank_transfer" enum value has no variant
	spec.Variants = spec.Variants[:1] // only CardPayment
	st := BuildSymbolTable(spec)
	findings := CheckSumTypes(spec, st)

	r16 := findingsWithRule(findings, "RULE-16")
	if len(r16) == 0 {
		t.Fatal("expected RULE-16 for missing BankTransfer variant declaration with snake_case enum")
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

func TestCheckSumTypes_RULE19_BaseEntityCreationSnakeCase(t *testing.T) {
	spec := sumTypeSpecSnakeCase()
	// Create base entity "Payment" instead of variant "CardPayment"
	spec.Rules[0].Ensures[0].Entity = "Payment"
	st := BuildSymbolTable(spec)
	findings := CheckSumTypes(spec, st)

	r19 := findingsWithRule(findings, "RULE-19")
	if len(r19) == 0 {
		t.Fatal("expected RULE-19 for base entity creation with snake_case discriminator")
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

// TestCheckSumTypes_NonDiscriminatorEnum verifies that an inline_enum whose values
// don't correspond to variant names is NOT treated as a discriminator.
func TestCheckSumTypes_NonDiscriminatorEnum(t *testing.T) {
	spec := &ast.Spec{
		File: "test.allium.json",
		Entities: []ast.Entity{
			{Name: "Task", Fields: []ast.Field{
				{Name: "title", Type: ast.FieldType{Kind: "primitive", Value: "String"}},
				{Name: "status", Type: ast.FieldType{Kind: "inline_enum", Values: []string{"open", "closed"}}},
			}},
		},
		Variants: []ast.Variant{
			{Name: "PriorityTask", BaseEntity: "Task", Fields: []ast.Field{
				{Name: "priority", Type: ast.FieldType{Kind: "primitive", Value: "Integer"}},
			}},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckSumTypes(spec, st)

	// PriorityTask doesn't match any enum value → no discriminator detected
	// → RULE-17 fires: variant extends entity with no discriminator
	r17 := findingsWithRule(findings, "RULE-17")
	if len(r17) == 0 {
		t.Fatal("expected RULE-17: inline_enum 'status' values don't match variant name 'PriorityTask'")
	}
}

func TestIsDiscriminatorField(t *testing.T) {
	tests := []struct {
		enumValues   []string
		variantNames []string
		want         bool
	}{
		{[]string{"Branch", "Leaf"}, []string{"Branch", "Leaf"}, true},              // direct match
		{[]string{"branch", "leaf"}, []string{"Branch", "Leaf"}, true},              // snake_case conversion
		{[]string{"card_payment", "bank_transfer"}, []string{"CardPayment", "BankTransfer"}, true},
		{[]string{"active", "inactive"}, []string{"PriorityTask"}, false},           // no correspondence
		{[]string{"open", "closed"}, []string{"PremiumUser"}, false},                // no correspondence
		{[]string{}, []string{"Branch"}, false},                                     // empty enum
		{[]string{"Branch"}, []string{}, false},                                     // no variants
		{[]string{"branch", "leaf"}, []string{"Branch", "Leaf", "Stem"}, true},      // partial match (any variant matches)
		{[]string{"A"}, []string{"A"}, true},                                        // single value
	}
	for _, tt := range tests {
		got := isDiscriminatorField(tt.enumValues, tt.variantNames)
		if got != tt.want {
			t.Errorf("isDiscriminatorField(%v, %v) = %v, want %v", tt.enumValues, tt.variantNames, got, tt.want)
		}
	}
}

func TestSnakeToPascal(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"branch", "Branch"},
		{"card_payment", "CardPayment"},
		{"bank_transfer", "BankTransfer"},
		{"a", "A"},
		{"premium_user", "PremiumUser"},
		{"Branch", "Branch"},               // already PascalCase (single word)
		{"already_PascalCase", "AlreadyPascalCase"}, // mixed: each segment gets first char uppercased
	}
	for _, tt := range tests {
		got := snakeToPascal(tt.input)
		if got != tt.want {
			t.Errorf("snakeToPascal(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestContainsVariantName(t *testing.T) {
	tests := []struct {
		enumValues  []string
		variantName string
		want        bool
	}{
		{[]string{"Branch", "Leaf"}, "Branch", true},
		{[]string{"branch", "leaf"}, "Branch", true},
		{[]string{"card_payment", "bank_transfer"}, "CardPayment", true},
		{[]string{"active", "inactive"}, "PriorityTask", false},
		{[]string{}, "Branch", false},
	}
	for _, tt := range tests {
		got := containsVariantName(tt.enumValues, tt.variantName)
		if got != tt.want {
			t.Errorf("containsVariantName(%v, %q) = %v, want %v", tt.enumValues, tt.variantName, got, tt.want)
		}
	}
}
