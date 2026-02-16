// Package semantic implements semantic analysis passes for Allium specifications.
package semantic

import (
	"github.com/foundry-zero/allium/internal/ast"
)

// SymbolTable indexes all named declarations in a specification for fast lookup.
type SymbolTable struct {
	Entities         map[string]*ast.Entity
	ExternalEntities map[string]*ast.ExternalEntity
	Rules            map[string]*ast.Rule
	Triggers         map[string][]*ast.Rule // multiple rules can share a trigger name
	Actors           map[string]*ast.Actor
	Surfaces         map[string]*ast.Surface
	Config           map[string]*ast.ConfigParam
	Given            map[string]*ast.GivenBinding
	Enumerations     map[string]*ast.Enumeration
	Variants         map[string]*ast.Variant
	UseDeclarations  map[string]*ast.UseDeclaration
	ValueTypes       map[string]*ast.ValueType
}

// BuildSymbolTable constructs a SymbolTable from a parsed specification.
// It populates all lookup maps by iterating through each declaration kind.
func BuildSymbolTable(spec *ast.Spec) *SymbolTable {
	st := &SymbolTable{
		Entities:         make(map[string]*ast.Entity, len(spec.Entities)),
		ExternalEntities: make(map[string]*ast.ExternalEntity, len(spec.ExternalEntities)),
		Rules:            make(map[string]*ast.Rule, len(spec.Rules)),
		Triggers:         make(map[string][]*ast.Rule),
		Actors:           make(map[string]*ast.Actor, len(spec.Actors)),
		Surfaces:         make(map[string]*ast.Surface, len(spec.Surfaces)),
		Config:           make(map[string]*ast.ConfigParam, len(spec.Config)),
		Given:            make(map[string]*ast.GivenBinding, len(spec.Given)),
		Enumerations:     make(map[string]*ast.Enumeration, len(spec.Enumerations)),
		Variants:         make(map[string]*ast.Variant, len(spec.Variants)),
		UseDeclarations:  make(map[string]*ast.UseDeclaration, len(spec.UseDeclarations)),
		ValueTypes:       make(map[string]*ast.ValueType, len(spec.ValueTypes)),
	}

	for i := range spec.Entities {
		st.Entities[spec.Entities[i].Name] = &spec.Entities[i]
	}
	for i := range spec.ExternalEntities {
		st.ExternalEntities[spec.ExternalEntities[i].Name] = &spec.ExternalEntities[i]
	}
	for i := range spec.Rules {
		r := &spec.Rules[i]
		st.Rules[r.Name] = r
		triggerName := triggerKeyName(r)
		if triggerName != "" {
			st.Triggers[triggerName] = append(st.Triggers[triggerName], r)
		}
	}
	for i := range spec.Actors {
		st.Actors[spec.Actors[i].Name] = &spec.Actors[i]
	}
	for i := range spec.Surfaces {
		st.Surfaces[spec.Surfaces[i].Name] = &spec.Surfaces[i]
	}
	for i := range spec.Config {
		st.Config[spec.Config[i].Name] = &spec.Config[i]
	}
	for i := range spec.Given {
		st.Given[spec.Given[i].Name] = &spec.Given[i]
	}
	for i := range spec.Enumerations {
		st.Enumerations[spec.Enumerations[i].Name] = &spec.Enumerations[i]
	}
	for i := range spec.Variants {
		st.Variants[spec.Variants[i].Name] = &spec.Variants[i]
	}
	for i := range spec.UseDeclarations {
		st.UseDeclarations[spec.UseDeclarations[i].Alias] = &spec.UseDeclarations[i]
	}
	for i := range spec.ValueTypes {
		st.ValueTypes[spec.ValueTypes[i].Name] = &spec.ValueTypes[i]
	}

	return st
}

// triggerKeyName returns the trigger name used for grouping rules.
// For external_stimulus and chained triggers this is the trigger name;
// for other kinds we return empty (they are not grouped by trigger name).
func triggerKeyName(r *ast.Rule) string {
	switch r.Trigger.Kind {
	case "external_stimulus", "chained":
		return r.Trigger.Name
	default:
		return ""
	}
}

// LookupEntity returns the entity with the given name, or nil if not found.
func (st *SymbolTable) LookupEntity(name string) *ast.Entity {
	return st.Entities[name]
}

// LookupExternalEntity returns the external entity with the given name, or nil.
func (st *SymbolTable) LookupExternalEntity(name string) *ast.ExternalEntity {
	return st.ExternalEntities[name]
}

// LookupAnyEntity returns true if name matches an entity, external entity,
// variant, or use declaration (imported type).
func (st *SymbolTable) LookupAnyEntity(name string) bool {
	if _, ok := st.Entities[name]; ok {
		return true
	}
	if _, ok := st.ExternalEntities[name]; ok {
		return true
	}
	if _, ok := st.Variants[name]; ok {
		return true
	}
	if _, ok := st.UseDeclarations[name]; ok {
		return true
	}
	return false
}

// LookupRule returns the rule with the given name, or nil.
func (st *SymbolTable) LookupRule(name string) *ast.Rule {
	return st.Rules[name]
}

// LookupTrigger returns all rules sharing the given trigger name.
func (st *SymbolTable) LookupTrigger(name string) []*ast.Rule {
	return st.Triggers[name]
}

// LookupActor returns the actor with the given name, or nil.
func (st *SymbolTable) LookupActor(name string) *ast.Actor {
	return st.Actors[name]
}

// LookupSurface returns the surface with the given name, or nil.
func (st *SymbolTable) LookupSurface(name string) *ast.Surface {
	return st.Surfaces[name]
}

// LookupConfig returns the config parameter with the given name, or nil.
func (st *SymbolTable) LookupConfig(name string) *ast.ConfigParam {
	return st.Config[name]
}

// LookupGiven returns the given binding with the given name, or nil.
func (st *SymbolTable) LookupGiven(name string) *ast.GivenBinding {
	return st.Given[name]
}

// LookupEnumeration returns the enumeration with the given name, or nil.
func (st *SymbolTable) LookupEnumeration(name string) *ast.Enumeration {
	return st.Enumerations[name]
}

// LookupVariant returns the variant with the given name, or nil.
func (st *SymbolTable) LookupVariant(name string) *ast.Variant {
	return st.Variants[name]
}

// LookupUseDeclaration returns the use declaration with the given alias, or nil.
func (st *SymbolTable) LookupUseDeclaration(name string) *ast.UseDeclaration {
	return st.UseDeclarations[name]
}

// LookupValueType returns the value type with the given name, or nil.
func (st *SymbolTable) LookupValueType(name string) *ast.ValueType {
	return st.ValueTypes[name]
}

// LookupType returns true if name matches any type-like declaration:
// entity, external entity, variant, use declaration, value type, or enumeration.
func (st *SymbolTable) LookupType(name string) bool {
	if st.LookupAnyEntity(name) {
		return true
	}
	if _, ok := st.ValueTypes[name]; ok {
		return true
	}
	if _, ok := st.Enumerations[name]; ok {
		return true
	}
	return false
}
