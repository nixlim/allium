package semantic

import (
	"testing"

	"github.com/foundry-zero/allium/internal/ast"
	"github.com/foundry-zero/allium/internal/report"
)

// cleanSpec returns a valid spec with no reference errors.
func cleanSpec() *ast.Spec {
	return &ast.Spec{
		Version: "0.4.0",
		File:    "test.allium.json",
		Entities: []ast.Entity{
			{
				Name: "Account",
				Fields: []ast.Field{
					{Name: "owner", Type: ast.FieldType{Kind: "entity_ref", Entity: "User"}},
					{Name: "status", Type: ast.FieldType{Kind: "named_enum", Name: "AccountStatus"}},
				},
				Relationships: []ast.Relationship{
					{Name: "transactions", TargetEntity: "Transaction", ForeignKey: "account_id", Cardinality: "many"},
				},
			},
			{Name: "User"},
			{Name: "Transaction"},
		},
		Enumerations: []ast.Enumeration{
			{Name: "AccountStatus", Values: []string{"active", "suspended"}},
		},
		Given: []ast.GivenBinding{
			{Name: "current_user", Type: ast.FieldType{Kind: "entity_ref", Entity: "User"}},
		},
		Config: []ast.ConfigParam{
			{Name: "max_retries", Type: ast.FieldType{Kind: "primitive", Value: "Integer"}},
		},
		Rules: []ast.Rule{
			{
				Name:    "CreateAccount",
				Trigger: ast.Trigger{Kind: "external_stimulus", Name: "create_account"},
			},
		},
		Actors: []ast.Actor{
			{Name: "EndUser", IdentifiedBy: ast.IdentifiedBy{Entity: "User"}},
		},
		Surfaces: []ast.Surface{
			{
				Name:   "Dashboard",
				Facing: ast.FacingClause{Binding: "user", Type: "EndUser"},
				Context: &ast.ContextClause{
					Binding: "acct",
					Type:    "Account",
				},
				Provides: []ast.ProvidesItem{
					{Kind: "action", Trigger: "create_account"},
				},
				Related: []ast.RelatedItem{},
			},
		},
	}
}

func findingWithRule(findings []report.Finding, rule string) *report.Finding {
	for i, f := range findings {
		if f.Rule == rule {
			return &findings[i]
		}
	}
	return nil
}

func findingsWithRule(findings []report.Finding, rule string) []report.Finding {
	var result []report.Finding
	for _, f := range findings {
		if f.Rule == rule {
			result = append(result, f)
		}
	}
	return result
}

func TestCheckReferences_CleanSpec(t *testing.T) {
	spec := cleanSpec()
	st := BuildSymbolTable(spec)
	findings := CheckReferences(spec, st)

	if len(findings) > 0 {
		for _, f := range findings {
			t.Errorf("unexpected finding: [%s] %s at %s", f.Rule, f.Message, f.Location.Path)
		}
	}
}

func TestCheckReferences_RULE01_EntityRef(t *testing.T) {
	spec := cleanSpec()
	spec.Entities[0].Fields[0].Type = ast.FieldType{Kind: "entity_ref", Entity: "FooBar"}
	st := BuildSymbolTable(spec)
	findings := CheckReferences(spec, st)

	f := findingWithRule(findings, "RULE-01")
	if f == nil {
		t.Fatal("expected RULE-01 finding for undeclared entity_ref")
	}
	if f.Severity != report.SeverityError {
		t.Errorf("severity = %v, want error", f.Severity)
	}
	if f.Location.Path != "$.entities[0].fields[0].type" {
		t.Errorf("path = %q", f.Location.Path)
	}
}

func TestCheckReferences_RULE01_NamedEnum(t *testing.T) {
	spec := cleanSpec()
	spec.Entities[0].Fields[1].Type = ast.FieldType{Kind: "named_enum", Name: "NoSuchEnum"}
	st := BuildSymbolTable(spec)
	findings := CheckReferences(spec, st)

	f := findingWithRule(findings, "RULE-01")
	if f == nil {
		t.Fatal("expected RULE-01 finding for undeclared named_enum")
	}
}

func TestCheckReferences_RULE01_OptionalInner(t *testing.T) {
	spec := cleanSpec()
	spec.Entities[0].Fields[0].Type = ast.FieldType{
		Kind:  "optional",
		Inner: &ast.FieldType{Kind: "entity_ref", Entity: "Missing"},
	}
	st := BuildSymbolTable(spec)
	findings := CheckReferences(spec, st)

	f := findingWithRule(findings, "RULE-01")
	if f == nil {
		t.Fatal("expected RULE-01 for optional inner entity_ref")
	}
	if f.Location.Path != "$.entities[0].fields[0].type.inner" {
		t.Errorf("path = %q", f.Location.Path)
	}
}

func TestCheckReferences_RULE01_SetElement(t *testing.T) {
	spec := cleanSpec()
	spec.Entities[0].Fields[0].Type = ast.FieldType{
		Kind:    "set",
		Element: &ast.FieldType{Kind: "entity_ref", Entity: "Gone"},
	}
	st := BuildSymbolTable(spec)
	findings := CheckReferences(spec, st)

	f := findingWithRule(findings, "RULE-01")
	if f == nil {
		t.Fatal("expected RULE-01 for set element entity_ref")
	}
}

func TestCheckReferences_RULE01_ExternalEntityFields(t *testing.T) {
	spec := cleanSpec()
	spec.ExternalEntities = []ast.ExternalEntity{
		{Name: "ExtSvc", Fields: []ast.Field{
			{Name: "ref", Type: ast.FieldType{Kind: "entity_ref", Entity: "Phantom"}},
		}},
	}
	st := BuildSymbolTable(spec)
	findings := CheckReferences(spec, st)

	f := findingWithRule(findings, "RULE-01")
	if f == nil {
		t.Fatal("expected RULE-01 for external entity field ref")
	}
}

func TestCheckReferences_RULE01_ValueTypeFields(t *testing.T) {
	spec := cleanSpec()
	spec.ValueTypes = []ast.ValueType{
		{Name: "Money", Fields: []ast.Field{
			{Name: "currency", Type: ast.FieldType{Kind: "named_enum", Name: "Ghost"}},
		}},
	}
	st := BuildSymbolTable(spec)
	findings := CheckReferences(spec, st)

	f := findingWithRule(findings, "RULE-01")
	if f == nil {
		t.Fatal("expected RULE-01 for value type field named_enum")
	}
}

func TestCheckReferences_RULE01_VariantFields(t *testing.T) {
	spec := cleanSpec()
	spec.Variants = []ast.Variant{
		{Name: "Premium", BaseEntity: "Account", Fields: []ast.Field{
			{Name: "tier", Type: ast.FieldType{Kind: "entity_ref", Entity: "Nope"}},
		}},
	}
	st := BuildSymbolTable(spec)
	findings := CheckReferences(spec, st)

	f := findingWithRule(findings, "RULE-01")
	if f == nil {
		t.Fatal("expected RULE-01 for variant field entity_ref")
	}
}

func TestCheckReferences_RULE01_ConfigFieldType(t *testing.T) {
	spec := cleanSpec()
	spec.Config = []ast.ConfigParam{
		{Name: "ref_param", Type: ast.FieldType{Kind: "entity_ref", Entity: "Missing"}},
	}
	st := BuildSymbolTable(spec)
	findings := CheckReferences(spec, st)

	f := findingWithRule(findings, "RULE-01")
	if f == nil {
		t.Fatal("expected RULE-01 for config field entity_ref")
	}
}

func TestCheckReferences_RULE01_ResolvesToVariant(t *testing.T) {
	spec := cleanSpec()
	spec.Variants = []ast.Variant{{Name: "PremiumAccount", BaseEntity: "Account"}}
	spec.Entities[0].Fields[0].Type = ast.FieldType{Kind: "entity_ref", Entity: "PremiumAccount"}
	st := BuildSymbolTable(spec)
	findings := CheckReferences(spec, st)

	r01 := findingsWithRule(findings, "RULE-01")
	if len(r01) > 0 {
		t.Errorf("entity_ref to variant should resolve, got %d RULE-01 findings", len(r01))
	}
}

func TestCheckReferences_RULE01_ResolvesToUseDecl(t *testing.T) {
	spec := cleanSpec()
	spec.UseDeclarations = []ast.UseDeclaration{{Coordinate: "auth/v1", Alias: "AuthUser"}}
	spec.Entities[0].Fields[0].Type = ast.FieldType{Kind: "entity_ref", Entity: "AuthUser"}
	st := BuildSymbolTable(spec)
	findings := CheckReferences(spec, st)

	r01 := findingsWithRule(findings, "RULE-01")
	if len(r01) > 0 {
		t.Error("entity_ref to use declaration should resolve")
	}
}

func TestCheckReferences_RULE03(t *testing.T) {
	spec := cleanSpec()
	spec.Entities[0].Relationships[0].TargetEntity = "NonExistent"
	st := BuildSymbolTable(spec)
	findings := CheckReferences(spec, st)

	f := findingWithRule(findings, "RULE-03")
	if f == nil {
		t.Fatal("expected RULE-03 finding")
	}
	if f.Severity != report.SeverityError {
		t.Errorf("severity = %v, want error", f.Severity)
	}
}

func TestCheckReferences_RULE22_EntityRef(t *testing.T) {
	spec := cleanSpec()
	spec.Given[0].Type = ast.FieldType{Kind: "entity_ref", Entity: "Unknown"}
	st := BuildSymbolTable(spec)
	findings := CheckReferences(spec, st)

	f := findingWithRule(findings, "RULE-22")
	if f == nil {
		t.Fatal("expected RULE-22 finding for undeclared given type")
	}
}

func TestCheckReferences_RULE22_NamedEnum(t *testing.T) {
	spec := cleanSpec()
	spec.Given[0].Type = ast.FieldType{Kind: "named_enum", Name: "Missing"}
	st := BuildSymbolTable(spec)
	findings := CheckReferences(spec, st)

	f := findingWithRule(findings, "RULE-22")
	if f == nil {
		t.Fatal("expected RULE-22 for undeclared given enumeration")
	}
}

func TestCheckReferences_RULE22_Primitive_NoError(t *testing.T) {
	spec := cleanSpec()
	spec.Given[0].Type = ast.FieldType{Kind: "primitive", Value: "String"}
	st := BuildSymbolTable(spec)
	findings := CheckReferences(spec, st)

	r22 := findingsWithRule(findings, "RULE-22")
	if len(r22) > 0 {
		t.Error("primitive given type should not trigger RULE-22")
	}
}

func TestCheckReferences_RULE28_FacingType(t *testing.T) {
	spec := cleanSpec()
	spec.Surfaces[0].Facing.Type = "UnknownActor"
	st := BuildSymbolTable(spec)
	findings := CheckReferences(spec, st)

	f := findingWithRule(findings, "RULE-28")
	if f == nil {
		t.Fatal("expected RULE-28 for undeclared facing type")
	}
}

func TestCheckReferences_RULE28_ContextType(t *testing.T) {
	spec := cleanSpec()
	spec.Surfaces[0].Context.Type = "MissingEntity"
	st := BuildSymbolTable(spec)
	findings := CheckReferences(spec, st)

	r28 := findingsWithRule(findings, "RULE-28")
	if len(r28) == 0 {
		t.Fatal("expected RULE-28 for undeclared context type")
	}
}

func TestCheckReferences_RULE28_ActorResolves(t *testing.T) {
	spec := cleanSpec()
	// Facing type "EndUser" is an actor — should resolve
	st := BuildSymbolTable(spec)
	findings := CheckReferences(spec, st)

	r28 := findingsWithRule(findings, "RULE-28")
	if len(r28) > 0 {
		t.Errorf("facing type matching actor should resolve, got %d RULE-28 findings", len(r28))
	}
}

func TestCheckReferences_RULE30(t *testing.T) {
	spec := cleanSpec()
	spec.Surfaces[0].Provides[0].Trigger = "non_existent_trigger"
	st := BuildSymbolTable(spec)
	findings := CheckReferences(spec, st)

	f := findingWithRule(findings, "RULE-30")
	if f == nil {
		t.Fatal("expected RULE-30 for undeclared provides trigger")
	}
}

func TestCheckReferences_RULE30_ForEachNested(t *testing.T) {
	spec := cleanSpec()
	spec.Surfaces[0].Provides = []ast.ProvidesItem{
		{
			Kind: "for_each",
			Items: []ast.ProvidesItem{
				{Kind: "action", Trigger: "missing_trigger"},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckReferences(spec, st)

	f := findingWithRule(findings, "RULE-30")
	if f == nil {
		t.Fatal("expected RULE-30 for nested for_each trigger")
	}
}

func TestCheckReferences_RULE31(t *testing.T) {
	spec := cleanSpec()
	spec.Surfaces[0].Related = []ast.RelatedItem{
		{Surface: "MissingSurface"},
	}
	st := BuildSymbolTable(spec)
	findings := CheckReferences(spec, st)

	f := findingWithRule(findings, "RULE-31")
	if f == nil {
		t.Fatal("expected RULE-31 for undeclared related surface")
	}
}

func TestCheckReferences_RULE31_ValidRelated(t *testing.T) {
	spec := cleanSpec()
	spec.Surfaces = append(spec.Surfaces, ast.Surface{
		Name:   "Settings",
		Facing: ast.FacingClause{Binding: "user", Type: "EndUser"},
	})
	spec.Surfaces[0].Related = []ast.RelatedItem{
		{Surface: "Settings"},
	}
	st := BuildSymbolTable(spec)
	findings := CheckReferences(spec, st)

	r31 := findingsWithRule(findings, "RULE-31")
	if len(r31) > 0 {
		t.Error("valid related surface should not trigger RULE-31")
	}
}

func TestCheckReferences_RULE35_EmptyCoordinate(t *testing.T) {
	spec := cleanSpec()
	spec.UseDeclarations = []ast.UseDeclaration{
		{Coordinate: "", Alias: "Bad"},
	}
	st := BuildSymbolTable(spec)
	findings := CheckReferences(spec, st)

	f := findingWithRule(findings, "RULE-35")
	if f == nil {
		t.Fatal("expected RULE-35 for empty use declaration coordinate")
	}
}

func TestCheckReferences_RULE27_UndeclaredConfigRef(t *testing.T) {
	spec := cleanSpec()
	// Add a rule with an expression that references config.nonexistent_param
	spec.Rules[0].Requires = []ast.Expression{
		{
			Kind: "comparison",
			Left: &ast.Expression{
				Kind:   "field_access",
				Field:  "nonexistent_param",
				Object: &ast.Expression{Kind: "field_access", Field: "config"},
			},
			Right: &ast.Expression{Kind: "literal", Type: "integer"},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckReferences(spec, st)

	f := findingWithRule(findings, "RULE-27")
	if f == nil {
		t.Fatal("expected RULE-27 for undeclared config parameter reference")
	}
	if f.Severity != report.SeverityError {
		t.Errorf("severity = %v, want error", f.Severity)
	}
}

func TestCheckReferences_RULE27_ValidConfigRef(t *testing.T) {
	spec := cleanSpec()
	// Reference the declared config param "max_retries"
	spec.Rules[0].Requires = []ast.Expression{
		{
			Kind: "comparison",
			Left: &ast.Expression{
				Kind:   "field_access",
				Field:  "max_retries",
				Object: &ast.Expression{Kind: "field_access", Field: "config"},
			},
			Right: &ast.Expression{Kind: "literal", Type: "integer"},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckReferences(spec, st)

	r27 := findingsWithRule(findings, "RULE-27")
	if len(r27) > 0 {
		t.Errorf("valid config reference should not trigger RULE-27, got %d findings", len(r27))
	}
}

func TestCheckReferences_RULE27_InDerivedValue(t *testing.T) {
	spec := cleanSpec()
	// Add derived value with undeclared config ref
	spec.Entities[0].DerivedValues = []ast.DerivedValue{
		{
			Name: "computed",
			Expression: &ast.Expression{
				Kind:   "field_access",
				Field:  "missing_config",
				Object: &ast.Expression{Kind: "field_access", Field: "config"},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckReferences(spec, st)

	f := findingWithRule(findings, "RULE-27")
	if f == nil {
		t.Fatal("expected RULE-27 for config reference in derived value")
	}
}

func TestCheckReferences_RULE27_InLetBinding(t *testing.T) {
	spec := cleanSpec()
	// Add let binding in rule with undeclared config ref
	spec.Rules[0].LetBindings = []ast.LetBinding{
		{
			Name: "threshold",
			Expression: &ast.Expression{
				Kind:   "field_access",
				Field:  "no_such_param",
				Object: &ast.Expression{Kind: "field_access", Field: "config"},
			},
		},
	}
	st := BuildSymbolTable(spec)
	findings := CheckReferences(spec, st)

	f := findingWithRule(findings, "RULE-27")
	if f == nil {
		t.Fatal("expected RULE-27 for config reference in let binding")
	}
}

func TestCheckReferences_MultipleErrors(t *testing.T) {
	spec := cleanSpec()
	// Break multiple things
	spec.Entities[0].Fields[0].Type = ast.FieldType{Kind: "entity_ref", Entity: "Missing1"}
	spec.Entities[0].Relationships[0].TargetEntity = "Missing2"
	spec.Surfaces[0].Facing.Type = "Missing3"

	st := BuildSymbolTable(spec)
	findings := CheckReferences(spec, st)

	if len(findings) < 3 {
		t.Errorf("expected at least 3 findings, got %d", len(findings))
		for _, f := range findings {
			t.Logf("  [%s] %s", f.Rule, f.Message)
		}
	}

	rules := map[string]bool{}
	for _, f := range findings {
		rules[f.Rule] = true
	}
	for _, want := range []string{"RULE-01", "RULE-03", "RULE-28"} {
		if !rules[want] {
			t.Errorf("missing expected rule %s", want)
		}
	}
}
