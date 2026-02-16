// Package ast defines the Go types for deserializing Allium specification JSON AST files.
package ast

import "encoding/json"

// Spec is the top-level representation of an Allium specification file.
type Spec struct {
	Version          string           `json:"version"`
	File             string           `json:"file"`
	Metadata         Metadata         `json:"metadata"`
	UseDeclarations  []UseDeclaration `json:"use_declarations"`
	Given            []GivenBinding   `json:"given"`
	ExternalEntities []ExternalEntity `json:"external_entities"`
	ValueTypes       []ValueType      `json:"value_types"`
	Enumerations     []Enumeration    `json:"enumerations"`
	Entities         []Entity         `json:"entities"`
	Variants         []Variant        `json:"variants"`
	Config           []ConfigParam    `json:"config"`
	Defaults         []Default        `json:"defaults"`
	Rules            []Rule           `json:"rules"`
	Actors           []Actor          `json:"actors"`
	Surfaces         []Surface        `json:"surfaces"`
	Deferred         []Deferred       `json:"deferred"`
	OpenQuestions    []string         `json:"open_questions"`
}

// Metadata holds optional file-level metadata.
type Metadata struct {
	Scope       string `json:"scope,omitempty"`
	Description string `json:"description,omitempty"`
}

// UseDeclaration represents an imported external spec.
type UseDeclaration struct {
	Coordinate string `json:"coordinate"`
	Alias      string `json:"alias"`
}

// GivenBinding declares an entity instance a module operates on.
type GivenBinding struct {
	Name string    `json:"name"`
	Type FieldType `json:"type"`
}

// ExternalEntity is an entity managed outside this specification.
type ExternalEntity struct {
	Name   string  `json:"name"`
	Fields []Field `json:"fields"`
}

// ValueType is structured data without identity.
type ValueType struct {
	Name          string         `json:"name"`
	Fields        []Field        `json:"fields"`
	DerivedValues []DerivedValue `json:"derived_values,omitempty"`
}

// Enumeration is a named set of values.
type Enumeration struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

// Entity is a domain concept with identity and lifecycle.
type Entity struct {
	Name          string         `json:"name"`
	Fields        []Field        `json:"fields"`
	Relationships []Relationship `json:"relationships,omitempty"`
	Projections   []Projection   `json:"projections,omitempty"`
	DerivedValues []DerivedValue `json:"derived_values,omitempty"`
}

// Variant is one alternative in a sum type.
type Variant struct {
	Name       string  `json:"name"`
	BaseEntity string  `json:"base_entity"`
	Fields     []Field `json:"fields"`
}

// Field is a named typed value on an entity or value type.
type Field struct {
	Name string    `json:"name"`
	Type FieldType `json:"type"`
}

// FieldType represents the type of a field, discriminated by Kind.
// Kind is one of: primitive, entity_ref, inline_enum, named_enum, optional, set, list.
type FieldType struct {
	Kind    string     `json:"kind"`
	Value   string     `json:"value,omitempty"`   // primitive: "String", "Integer", etc.
	Entity  string     `json:"entity,omitempty"`  // entity_ref
	Values  []string   `json:"values,omitempty"`  // inline_enum
	Name    string     `json:"name,omitempty"`    // named_enum
	Inner   *FieldType `json:"inner,omitempty"`   // optional
	Element *FieldType `json:"element,omitempty"` // set, list
}

// Relationship navigates from one entity to related entities.
type Relationship struct {
	Name         string `json:"name"`
	TargetEntity string `json:"target_entity"`
	ForeignKey   string `json:"foreign_key"`
	Cardinality  string `json:"cardinality"` // "one" or "many"
}

// Projection is a filtered view of a relationship.
type Projection struct {
	Name      string      `json:"name"`
	Source    string      `json:"source"`
	Condition *Expression `json:"condition"`
	Mapping   string      `json:"mapping,omitempty"`
}

// DerivedValue is a computed value based on other fields.
type DerivedValue struct {
	Name       string      `json:"name"`
	Parameters []string    `json:"parameters,omitempty"`
	Expression *Expression `json:"expression"`
}

// ConfigParam is a configurable parameter with a default value.
type ConfigParam struct {
	Name         string      `json:"name"`
	Type         FieldType   `json:"type"`
	DefaultValue *Expression `json:"default_value"`
}

// Default is a named entity instance used as seed data.
type Default struct {
	Entity string                `json:"entity"`
	Name   string                `json:"name"`
	Fields map[string]Expression `json:"fields"`
}

// Rule defines behaviour triggered by some condition.
type Rule struct {
	Name        string          `json:"name"`
	Trigger     Trigger         `json:"trigger"`
	ForClause   *ForClause      `json:"for_clause,omitempty"`
	LetBindings []LetBinding    `json:"let_bindings,omitempty"`
	Requires    []Expression    `json:"requires,omitempty"`
	Ensures     []EnsuresClause `json:"ensures"`
}

// Trigger is the condition that causes a rule to fire.
// Kind is one of: external_stimulus, state_transition, state_becomes,
// temporal, derived_condition, entity_creation, chained.
type Trigger struct {
	Kind       string           `json:"kind"`
	Name       string           `json:"name,omitempty"`       // external_stimulus, chained
	Parameters []TriggerParam   `json:"parameters,omitempty"` // external_stimulus, chained
	Binding    string           `json:"binding,omitempty"`    // state_transition, state_becomes, temporal, derived_condition, entity_creation
	Entity     string           `json:"entity,omitempty"`     // all binding triggers
	Field      string           `json:"field,omitempty"`      // state_transition, state_becomes, derived_condition
	ToValue    string           `json:"to_value,omitempty"`   // state_transition
	Value      string           `json:"value,omitempty"`      // state_becomes
	Condition  *Expression      `json:"condition,omitempty"`  // temporal
}

// TriggerParam is a named parameter of a trigger.
type TriggerParam struct {
	Name     string `json:"name"`
	Optional bool   `json:"optional,omitempty"`
}

// ForClause applies a rule body once per element in a collection.
type ForClause struct {
	Binding    string      `json:"binding"`
	Collection *Expression `json:"collection"`
	Condition  *Expression `json:"condition,omitempty"`
}

// LetBinding introduces a local variable.
type LetBinding struct {
	Name       string      `json:"name"`
	Expression *Expression `json:"expression"`
}

// EnsuresClause describes what becomes true after a rule executes.
// Kind is one of: state_change, entity_creation, trigger_emission,
// entity_removal, conditional, iteration, let_binding, set_mutation.
type EnsuresClause struct {
	Kind string `json:"kind"`

	// state_change, entity_removal, set_mutation
	Target *Expression `json:"target,omitempty"`

	// state_change value (Expression), let_binding value (entity creation or Expression)
	Value json.RawMessage `json:"value,omitempty"`

	// entity_creation, trigger_emission
	Entity string `json:"entity,omitempty"`
	Name   string `json:"name,omitempty"`

	// entity_creation fields, trigger_emission arguments share the same structure
	Fields    map[string]Expression `json:"fields,omitempty"`
	Arguments map[string]Expression `json:"arguments,omitempty"`

	// conditional
	Condition *Expression     `json:"condition,omitempty"`
	Then      []EnsuresClause `json:"then,omitempty"`
	Else      []EnsuresClause `json:"else,omitempty"`

	// iteration, let_binding
	Binding    string          `json:"binding,omitempty"`
	Collection *Expression     `json:"collection,omitempty"`
	Body       []EnsuresClause `json:"body,omitempty"`

	// set_mutation
	Operation string `json:"operation,omitempty"`
}

// Expression represents a node in the expression tree.
// Kind is one of: field_access, literal, comparison, arithmetic, boolean_logic,
// function_call, collection_op, exists, not, null_coalesce, set_literal,
// membership, join_lookup, lambda.
type Expression struct {
	Kind string `json:"kind"`

	// field_access
	Object *Expression `json:"object,omitempty"` // null for root access in field_access
	Field  string      `json:"field,omitempty"`

	// literal
	Type     string          `json:"type,omitempty"`  // "string", "integer", "boolean", "null", "duration", "enum_value", "timestamp"
	LitValue json.RawMessage `json:"value,omitempty"` // actual literal value

	// comparison, arithmetic, boolean_logic, null_coalesce
	Operator string      `json:"operator,omitempty"`
	Left     *Expression `json:"left,omitempty"`
	Right    *Expression `json:"right,omitempty"`

	// function_call
	FuncName      string       `json:"name,omitempty"`
	FuncArguments []Expression `json:"arguments,omitempty"`

	// collection_op, membership
	Operation  string      `json:"operation,omitempty"`
	Collection *Expression `json:"collection,omitempty"`
	Lambda     *Expression `json:"lambda,omitempty"`
	Condition  *Expression `json:"condition,omitempty"`

	// exists
	Target *Expression `json:"target,omitempty"`

	// not
	Operand *Expression `json:"operand,omitempty"`

	// set_literal
	Elements []Expression `json:"elements,omitempty"`

	// membership
	Element *Expression `json:"element,omitempty"`

	// join_lookup
	Entity string                `json:"entity,omitempty"`
	Fields map[string]Expression `json:"fields,omitempty"`

	// lambda
	Parameter string      `json:"parameter,omitempty"`
	Body      *Expression `json:"body,omitempty"`
}

// Actor declares an entity type that can interact with surfaces.
type Actor struct {
	Name         string       `json:"name"`
	Within       string       `json:"within,omitempty"`
	IdentifiedBy IdentifiedBy `json:"identified_by"`
}

// IdentifiedBy specifies the entity type and condition that identifies an actor.
type IdentifiedBy struct {
	Entity    string      `json:"entity"`
	Condition *Expression `json:"condition"`
}

// Surface defines a contract at a boundary.
type Surface struct {
	Name        string          `json:"name"`
	Facing      FacingClause    `json:"facing"`
	Context     *ContextClause  `json:"context"`
	LetBindings []LetBinding    `json:"let_bindings,omitempty"`
	Exposes     []ExposesItem   `json:"exposes,omitempty"`
	Provides    []ProvidesItem  `json:"provides,omitempty"`
	Guarantees  []Guarantee     `json:"guarantees,omitempty"`
	Guidance    []string        `json:"guidance,omitempty"`
	Related     []RelatedItem   `json:"related,omitempty"`
	Timeout     []TimeoutItem   `json:"timeout,omitempty"`
}

// FacingClause names the external party on the other side of the boundary.
type FacingClause struct {
	Binding string `json:"binding"`
	Type    string `json:"type"`
}

// ContextClause binds a parametric scope for a surface.
type ContextClause struct {
	Binding   string      `json:"binding"`
	Type      string      `json:"type"`
	Condition *Expression `json:"condition,omitempty"`
}

// ExposesItem is a visible data item in a surface.
type ExposesItem struct {
	Expression *Expression `json:"expression"`
	When       *Expression `json:"when,omitempty"`
}

// ProvidesItem is an available operation in a surface.
// Kind is "action" or "for_each".
type ProvidesItem struct {
	Kind string `json:"kind"`

	// action
	Trigger   string            `json:"trigger,omitempty"`
	Arguments []ProvideArgument `json:"arguments,omitempty"`
	When      *Expression       `json:"when,omitempty"`

	// for_each
	Binding    string         `json:"binding,omitempty"`
	Collection *Expression    `json:"collection,omitempty"`
	Items      []ProvidesItem `json:"items,omitempty"`
}

// ProvideArgument is a named argument in a provides action.
type ProvideArgument struct {
	Name       string      `json:"name"`
	Expression *Expression `json:"expression,omitempty"`
}

// Guarantee is a constraint that must hold across a boundary.
type Guarantee struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// RelatedItem references an associated surface reachable from the current one.
type RelatedItem struct {
	Surface           string      `json:"surface"`
	ContextExpression *Expression `json:"context_expression"`
	When              *Expression `json:"when,omitempty"`
}

// TimeoutItem references a temporal rule within a surface's context.
type TimeoutItem struct {
	Rule string      `json:"rule"`
	When *Expression `json:"when,omitempty"`
}

// Deferred references a detailed specification defined elsewhere.
type Deferred struct {
	Name         string  `json:"name"`
	Method       string  `json:"method"`
	LocationHint *string `json:"location_hint"`
}
