package semantic

import (
	"testing"

	"github.com/foundry-zero/allium/internal/ast"
)

func makeTestSpec() *ast.Spec {
	return &ast.Spec{
		Version: "0.4.0",
		Entities: []ast.Entity{
			{Name: "Account", Fields: []ast.Field{{Name: "status", Type: ast.FieldType{Kind: "primitive", Value: "String"}}}},
			{Name: "Transaction"},
		},
		ExternalEntities: []ast.ExternalEntity{
			{Name: "PaymentGateway"},
		},
		ValueTypes: []ast.ValueType{
			{Name: "Money", Fields: []ast.Field{{Name: "amount", Type: ast.FieldType{Kind: "primitive", Value: "Integer"}}}},
		},
		Enumerations: []ast.Enumeration{
			{Name: "Currency", Values: []string{"USD", "EUR"}},
		},
		Variants: []ast.Variant{
			{Name: "PremiumAccount", BaseEntity: "Account"},
		},
		UseDeclarations: []ast.UseDeclaration{
			{Coordinate: "auth/v1", Alias: "Auth"},
		},
		Given: []ast.GivenBinding{
			{Name: "current_account", Type: ast.FieldType{Kind: "entity_ref", Entity: "Account"}},
		},
		Config: []ast.ConfigParam{
			{Name: "max_retries", Type: ast.FieldType{Kind: "primitive", Value: "Integer"}},
		},
		Rules: []ast.Rule{
			{Name: "CreateAccount", Trigger: ast.Trigger{Kind: "external_stimulus", Name: "create_account", Parameters: []ast.TriggerParam{{Name: "owner"}}}},
			{Name: "VerifyAccount", Trigger: ast.Trigger{Kind: "external_stimulus", Name: "create_account", Parameters: []ast.TriggerParam{{Name: "owner"}}}},
			{Name: "ExpireSession", Trigger: ast.Trigger{Kind: "temporal", Entity: "Session", Field: "expires_at"}},
			{Name: "ChainedRule", Trigger: ast.Trigger{Kind: "chained", Name: "after_create"}},
		},
		Actors: []ast.Actor{
			{Name: "EndUser", IdentifiedBy: ast.IdentifiedBy{Entity: "Account"}},
		},
		Surfaces: []ast.Surface{
			{Name: "AccountDashboard", Facing: ast.FacingClause{Binding: "user", Type: "EndUser"}},
		},
	}
}

func TestBuildSymbolTable(t *testing.T) {
	spec := makeTestSpec()
	st := BuildSymbolTable(spec)

	if st == nil {
		t.Fatal("BuildSymbolTable returned nil")
	}

	if len(st.Entities) != 2 {
		t.Errorf("Entities count = %d, want 2", len(st.Entities))
	}
	if len(st.ExternalEntities) != 1 {
		t.Errorf("ExternalEntities count = %d, want 1", len(st.ExternalEntities))
	}
	if len(st.Rules) != 4 {
		t.Errorf("Rules count = %d, want 4", len(st.Rules))
	}
	if len(st.Actors) != 1 {
		t.Errorf("Actors count = %d, want 1", len(st.Actors))
	}
	if len(st.Surfaces) != 1 {
		t.Errorf("Surfaces count = %d, want 1", len(st.Surfaces))
	}
	if len(st.Config) != 1 {
		t.Errorf("Config count = %d, want 1", len(st.Config))
	}
	if len(st.Given) != 1 {
		t.Errorf("Given count = %d, want 1", len(st.Given))
	}
	if len(st.Enumerations) != 1 {
		t.Errorf("Enumerations count = %d, want 1", len(st.Enumerations))
	}
	if len(st.Variants) != 1 {
		t.Errorf("Variants count = %d, want 1", len(st.Variants))
	}
	if len(st.UseDeclarations) != 1 {
		t.Errorf("UseDeclarations count = %d, want 1", len(st.UseDeclarations))
	}
	if len(st.ValueTypes) != 1 {
		t.Errorf("ValueTypes count = %d, want 1", len(st.ValueTypes))
	}
}

func TestLookupEntity(t *testing.T) {
	st := BuildSymbolTable(makeTestSpec())

	if e := st.LookupEntity("Account"); e == nil {
		t.Error("LookupEntity(Account) returned nil")
	} else if e.Name != "Account" {
		t.Errorf("LookupEntity(Account).Name = %q", e.Name)
	}

	if e := st.LookupEntity("Missing"); e != nil {
		t.Error("LookupEntity(Missing) should return nil")
	}
}

func TestLookupExternalEntity(t *testing.T) {
	st := BuildSymbolTable(makeTestSpec())

	if e := st.LookupExternalEntity("PaymentGateway"); e == nil {
		t.Error("LookupExternalEntity(PaymentGateway) returned nil")
	}
	if e := st.LookupExternalEntity("Missing"); e != nil {
		t.Error("LookupExternalEntity(Missing) should return nil")
	}
}

func TestLookupAnyEntity(t *testing.T) {
	st := BuildSymbolTable(makeTestSpec())

	// Regular entity
	if !st.LookupAnyEntity("Account") {
		t.Error("LookupAnyEntity(Account) = false")
	}
	// External entity
	if !st.LookupAnyEntity("PaymentGateway") {
		t.Error("LookupAnyEntity(PaymentGateway) = false")
	}
	// Variant
	if !st.LookupAnyEntity("PremiumAccount") {
		t.Error("LookupAnyEntity(PremiumAccount) = false")
	}
	// Use declaration
	if !st.LookupAnyEntity("Auth") {
		t.Error("LookupAnyEntity(Auth) = false")
	}
	// Missing
	if st.LookupAnyEntity("NoSuchEntity") {
		t.Error("LookupAnyEntity(NoSuchEntity) = true")
	}
}

func TestLookupRule(t *testing.T) {
	st := BuildSymbolTable(makeTestSpec())

	if r := st.LookupRule("CreateAccount"); r == nil {
		t.Error("LookupRule(CreateAccount) returned nil")
	}
	if r := st.LookupRule("Missing"); r != nil {
		t.Error("LookupRule(Missing) should return nil")
	}
}

func TestTriggerGrouping(t *testing.T) {
	st := BuildSymbolTable(makeTestSpec())

	// Two rules share trigger "create_account"
	rules := st.LookupTrigger("create_account")
	if len(rules) != 2 {
		t.Fatalf("LookupTrigger(create_account) = %d rules, want 2", len(rules))
	}

	// Chained trigger
	rules = st.LookupTrigger("after_create")
	if len(rules) != 1 {
		t.Fatalf("LookupTrigger(after_create) = %d rules, want 1", len(rules))
	}

	// Temporal triggers have no name key for grouping
	rules = st.LookupTrigger("")
	if len(rules) != 0 {
		t.Errorf("LookupTrigger('') = %d rules, want 0", len(rules))
	}

	// Non-existent trigger
	rules = st.LookupTrigger("no_such_trigger")
	if len(rules) != 0 {
		t.Errorf("LookupTrigger(no_such_trigger) = %d, want 0", len(rules))
	}
}

func TestLookupActor(t *testing.T) {
	st := BuildSymbolTable(makeTestSpec())

	if a := st.LookupActor("EndUser"); a == nil {
		t.Error("LookupActor(EndUser) returned nil")
	}
	if a := st.LookupActor("Missing"); a != nil {
		t.Error("LookupActor(Missing) should return nil")
	}
}

func TestLookupSurface(t *testing.T) {
	st := BuildSymbolTable(makeTestSpec())

	if s := st.LookupSurface("AccountDashboard"); s == nil {
		t.Error("LookupSurface(AccountDashboard) returned nil")
	}
	if s := st.LookupSurface("Missing"); s != nil {
		t.Error("LookupSurface(Missing) should return nil")
	}
}

func TestLookupConfig(t *testing.T) {
	st := BuildSymbolTable(makeTestSpec())

	if c := st.LookupConfig("max_retries"); c == nil {
		t.Error("LookupConfig(max_retries) returned nil")
	}
	if c := st.LookupConfig("Missing"); c != nil {
		t.Error("LookupConfig(Missing) should return nil")
	}
}

func TestLookupGiven(t *testing.T) {
	st := BuildSymbolTable(makeTestSpec())

	if g := st.LookupGiven("current_account"); g == nil {
		t.Error("LookupGiven(current_account) returned nil")
	}
	if g := st.LookupGiven("Missing"); g != nil {
		t.Error("LookupGiven(Missing) should return nil")
	}
}

func TestLookupEnumeration(t *testing.T) {
	st := BuildSymbolTable(makeTestSpec())

	if e := st.LookupEnumeration("Currency"); e == nil {
		t.Error("LookupEnumeration(Currency) returned nil")
	}
	if e := st.LookupEnumeration("Missing"); e != nil {
		t.Error("LookupEnumeration(Missing) should return nil")
	}
}

func TestLookupVariant(t *testing.T) {
	st := BuildSymbolTable(makeTestSpec())

	if v := st.LookupVariant("PremiumAccount"); v == nil {
		t.Error("LookupVariant(PremiumAccount) returned nil")
	}
	if v := st.LookupVariant("Missing"); v != nil {
		t.Error("LookupVariant(Missing) should return nil")
	}
}

func TestLookupUseDeclaration(t *testing.T) {
	st := BuildSymbolTable(makeTestSpec())

	if u := st.LookupUseDeclaration("Auth"); u == nil {
		t.Error("LookupUseDeclaration(Auth) returned nil")
	}
	if u := st.LookupUseDeclaration("Missing"); u != nil {
		t.Error("LookupUseDeclaration(Missing) should return nil")
	}
}

func TestLookupValueType(t *testing.T) {
	st := BuildSymbolTable(makeTestSpec())

	if v := st.LookupValueType("Money"); v == nil {
		t.Error("LookupValueType(Money) returned nil")
	}
	if v := st.LookupValueType("Missing"); v != nil {
		t.Error("LookupValueType(Missing) should return nil")
	}
}

func TestLookupType(t *testing.T) {
	st := BuildSymbolTable(makeTestSpec())

	// Entity
	if !st.LookupType("Account") {
		t.Error("LookupType(Account) = false, want true")
	}
	// External entity
	if !st.LookupType("PaymentGateway") {
		t.Error("LookupType(PaymentGateway) = false, want true")
	}
	// Variant
	if !st.LookupType("PremiumAccount") {
		t.Error("LookupType(PremiumAccount) = false, want true")
	}
	// Use declaration
	if !st.LookupType("Auth") {
		t.Error("LookupType(Auth) = false, want true")
	}
	// Value type
	if !st.LookupType("Money") {
		t.Error("LookupType(Money) = false, want true")
	}
	// Enumeration
	if !st.LookupType("Currency") {
		t.Error("LookupType(Currency) = false, want true")
	}
	// Missing
	if st.LookupType("NoSuchType") {
		t.Error("LookupType(NoSuchType) = true, want false")
	}
}

func TestEmptySpec(t *testing.T) {
	st := BuildSymbolTable(&ast.Spec{})

	if len(st.Entities) != 0 {
		t.Errorf("expected empty Entities map")
	}
	if len(st.Triggers) != 0 {
		t.Errorf("expected empty Triggers map")
	}
	if st.LookupEntity("anything") != nil {
		t.Error("lookup on empty table should return nil")
	}
}

func TestPointersAreStable(t *testing.T) {
	spec := makeTestSpec()
	st := BuildSymbolTable(spec)

	// Verify the symbol table points into the spec's slices
	e := st.LookupEntity("Account")
	if e == nil {
		t.Fatal("Account not found")
	}
	if e != &spec.Entities[0] {
		t.Error("symbol table entity pointer does not reference spec slice element")
	}
}
