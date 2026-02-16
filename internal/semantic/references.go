package semantic

import (
	"fmt"

	"github.com/foundry-zero/allium/internal/ast"
	"github.com/foundry-zero/allium/internal/report"
)

// CheckReferences verifies that all name references in the specification
// resolve to declared symbols. It checks rules:
//
//   - RULE-01: entity_ref types resolve to a declared entity/external/variant/import
//   - RULE-03: relationship target_entity resolves
//   - RULE-22: given binding type references resolve
//   - RULE-27: config parameter references in expressions resolve
//   - RULE-28: surface facing type resolves to entity or actor
//   - RULE-30: surface provides trigger resolves to a declared rule trigger
//   - RULE-31: surface related surface_name resolves
//   - RULE-35: use_declaration coordinate is noted (unresolvable cross-spec)
func CheckReferences(spec *ast.Spec, st *SymbolTable) []report.Finding {
	var findings []report.Finding

	// RULE-01: Check all entity_ref types across entities, external entities, value types, variants
	for i, e := range spec.Entities {
		for j, f := range e.Fields {
			findings = checkFieldTypeRefs(findings, spec, st, f.Type,
				fmt.Sprintf("$.entities[%d].fields[%d].type", i, j))
		}
		for j, dv := range e.DerivedValues {
			findings = checkExpressionConfigRefs(findings, st, dv.Expression,
				fmt.Sprintf("$.entities[%d].derived_values[%d].expression", i, j), spec.File)
		}
	}
	for i, e := range spec.ExternalEntities {
		for j, f := range e.Fields {
			findings = checkFieldTypeRefs(findings, spec, st, f.Type,
				fmt.Sprintf("$.external_entities[%d].fields[%d].type", i, j))
		}
	}
	for i, vt := range spec.ValueTypes {
		for j, f := range vt.Fields {
			findings = checkFieldTypeRefs(findings, spec, st, f.Type,
				fmt.Sprintf("$.value_types[%d].fields[%d].type", i, j))
		}
	}
	for i, v := range spec.Variants {
		for j, f := range v.Fields {
			findings = checkFieldTypeRefs(findings, spec, st, f.Type,
				fmt.Sprintf("$.variants[%d].fields[%d].type", i, j))
		}
	}
	for i, c := range spec.Config {
		findings = checkFieldTypeRefs(findings, spec, st, c.Type,
			fmt.Sprintf("$.config[%d].type", i))
	}

	// RULE-03: Check relationship target_entity references
	for i, e := range spec.Entities {
		for j, rel := range e.Relationships {
			if !st.LookupAnyEntity(rel.TargetEntity) {
				findings = append(findings, report.NewError(
					"RULE-03",
					fmt.Sprintf("Relationship '%s' target entity '%s' not declared", rel.Name, rel.TargetEntity),
					report.Location{File: spec.File, Path: fmt.Sprintf("$.entities[%d].relationships[%d].target_entity", i, j)},
				))
			}
		}
	}

	// RULE-22: Check given binding types
	for i, g := range spec.Given {
		findings = checkGivenTypeRef(findings, spec, st, g,
			fmt.Sprintf("$.given[%d].type", i))
	}

	// RULE-27: Check config references in rule expressions
	for i, r := range spec.Rules {
		findings = checkRuleConfigRefs(findings, spec, st, r, i)
	}

	// RULE-28: Check surface facing type
	for i, s := range spec.Surfaces {
		facingType := s.Facing.Type
		if !st.LookupAnyEntity(facingType) && st.LookupActor(facingType) == nil {
			findings = append(findings, report.NewError(
				"RULE-28",
				fmt.Sprintf("Surface '%s' facing type '%s' not declared as entity or actor", s.Name, facingType),
				report.Location{File: spec.File, Path: fmt.Sprintf("$.surfaces[%d].facing.type", i)},
			))
		}

		// Also check context type if present
		if s.Context != nil {
			ctxType := s.Context.Type
			if !st.LookupAnyEntity(ctxType) {
				findings = append(findings, report.NewError(
					"RULE-28",
					fmt.Sprintf("Surface '%s' context type '%s' not declared", s.Name, ctxType),
					report.Location{File: spec.File, Path: fmt.Sprintf("$.surfaces[%d].context.type", i)},
				))
			}
		}
	}

	// RULE-30: Check surface provides trigger names
	for i, s := range spec.Surfaces {
		for j, p := range s.Provides {
			findings = checkProvidesItemTrigger(findings, spec, st, p, s.Name,
				fmt.Sprintf("$.surfaces[%d].provides[%d]", i, j))
		}
	}

	// RULE-31: Check surface related surface names
	for i, s := range spec.Surfaces {
		for j, rel := range s.Related {
			if st.LookupSurface(rel.Surface) == nil {
				findings = append(findings, report.NewError(
					"RULE-31",
					fmt.Sprintf("Surface '%s' related surface '%s' not declared", s.Name, rel.Surface),
					report.Location{File: spec.File, Path: fmt.Sprintf("$.surfaces[%d].related[%d].surface", i, j)},
				))
			}
		}
	}

	// RULE-35: Use declarations — cross-spec resolution is beyond our scope,
	// but we note if the coordinate is empty (structural issue).
	for i, u := range spec.UseDeclarations {
		if u.Coordinate == "" {
			findings = append(findings, report.NewError(
				"RULE-35",
				fmt.Sprintf("Use declaration '%s' has empty coordinate", u.Alias),
				report.Location{File: spec.File, Path: fmt.Sprintf("$.use_declarations[%d].coordinate", i)},
			))
		}
	}

	return findings
}

// checkFieldTypeRefs recursively checks entity_ref, named_enum, optional, set, list types.
func checkFieldTypeRefs(findings []report.Finding, spec *ast.Spec, st *SymbolTable, ft ast.FieldType, path string) []report.Finding {
	switch ft.Kind {
	case "entity_ref":
		if !st.LookupAnyEntity(ft.Entity) {
			findings = append(findings, report.NewError(
				"RULE-01",
				fmt.Sprintf("Entity '%s' referenced but not declared", ft.Entity),
				report.Location{File: spec.File, Path: path},
			))
		}
	case "named_enum":
		if st.LookupEnumeration(ft.Name) == nil {
			findings = append(findings, report.NewError(
				"RULE-01",
				fmt.Sprintf("Enumeration '%s' referenced but not declared", ft.Name),
				report.Location{File: spec.File, Path: path},
			))
		}
	case "optional":
		if ft.Inner != nil {
			findings = checkFieldTypeRefs(findings, spec, st, *ft.Inner, path+".inner")
		}
	case "set", "list":
		if ft.Element != nil {
			findings = checkFieldTypeRefs(findings, spec, st, *ft.Element, path+".element")
		}
	}
	return findings
}

// checkGivenTypeRef checks that a given binding's type resolves.
func checkGivenTypeRef(findings []report.Finding, spec *ast.Spec, st *SymbolTable, g ast.GivenBinding, path string) []report.Finding {
	switch g.Type.Kind {
	case "entity_ref":
		if !st.LookupAnyEntity(g.Type.Entity) {
			findings = append(findings, report.NewError(
				"RULE-22",
				fmt.Sprintf("Given binding '%s' references undeclared entity '%s'", g.Name, g.Type.Entity),
				report.Location{File: spec.File, Path: path},
			))
		}
	case "named_enum":
		if st.LookupEnumeration(g.Type.Name) == nil {
			findings = append(findings, report.NewError(
				"RULE-22",
				fmt.Sprintf("Given binding '%s' references undeclared enumeration '%s'", g.Name, g.Type.Name),
				report.Location{File: spec.File, Path: path},
			))
		}
	default:
		// primitive and other types don't need reference checks
	}
	return findings
}

// checkRuleConfigRefs walks rule expressions looking for config parameter references.
func checkRuleConfigRefs(findings []report.Finding, spec *ast.Spec, st *SymbolTable, r ast.Rule, ruleIdx int) []report.Finding {
	basePath := fmt.Sprintf("$.rules[%d]", ruleIdx)

	// Check requires expressions
	for j, expr := range r.Requires {
		findings = checkExpressionConfigRefs(findings, st, &expr,
			fmt.Sprintf("%s.requires[%d]", basePath, j), spec.File)
	}

	// Check ensures clauses
	for j, ec := range r.Ensures {
		findings = checkEnsuresConfigRefs(findings, st, ec,
			fmt.Sprintf("%s.ensures[%d]", basePath, j), spec.File)
	}

	// Check let bindings
	for j, lb := range r.LetBindings {
		findings = checkExpressionConfigRefs(findings, st, lb.Expression,
			fmt.Sprintf("%s.let_bindings[%d].expression", basePath, j), spec.File)
	}

	return findings
}

// checkExpressionConfigRefs walks an expression tree looking for config references (RULE-27).
func checkExpressionConfigRefs(findings []report.Finding, st *SymbolTable, expr *ast.Expression, path string, file string) []report.Finding {
	if expr == nil {
		return findings
	}

	// A config reference is config.param_name: a field_access where the object is
	// a root field_access with field "config", and the outer field is the param name.
	if expr.Kind == "field_access" && expr.Object != nil &&
		expr.Object.Kind == "field_access" && expr.Object.Object == nil && expr.Object.Field == "config" {
		paramName := expr.Field
		if st.LookupConfig(paramName) == nil {
			findings = append(findings, report.NewError(
				"RULE-27",
				fmt.Sprintf("Config parameter '%s' referenced but not declared", paramName),
				report.Location{File: file, Path: path},
			))
		}
	}

	// Recurse into sub-expressions
	findings = checkExpressionConfigRefs(findings, st, expr.Object, path+".object", file)
	findings = checkExpressionConfigRefs(findings, st, expr.Left, path+".left", file)
	findings = checkExpressionConfigRefs(findings, st, expr.Right, path+".right", file)
	findings = checkExpressionConfigRefs(findings, st, expr.Target, path+".target", file)
	findings = checkExpressionConfigRefs(findings, st, expr.Operand, path+".operand", file)
	findings = checkExpressionConfigRefs(findings, st, expr.Collection, path+".collection", file)
	findings = checkExpressionConfigRefs(findings, st, expr.Lambda, path+".lambda", file)
	findings = checkExpressionConfigRefs(findings, st, expr.Condition, path+".condition", file)
	findings = checkExpressionConfigRefs(findings, st, expr.Body, path+".body", file)
	findings = checkExpressionConfigRefs(findings, st, expr.Element, path+".element", file)

	for j := range expr.FuncArguments {
		findings = checkExpressionConfigRefs(findings, st, &expr.FuncArguments[j],
			fmt.Sprintf("%s.arguments[%d]", path, j), file)
	}
	for j := range expr.Elements {
		findings = checkExpressionConfigRefs(findings, st, &expr.Elements[j],
			fmt.Sprintf("%s.elements[%d]", path, j), file)
	}

	return findings
}

// checkEnsuresConfigRefs walks an ensures clause tree for config references.
func checkEnsuresConfigRefs(findings []report.Finding, st *SymbolTable, ec ast.EnsuresClause, path string, file string) []report.Finding {
	findings = checkExpressionConfigRefs(findings, st, ec.Target, path+".target", file)
	findings = checkExpressionConfigRefs(findings, st, ec.Condition, path+".condition", file)
	findings = checkExpressionConfigRefs(findings, st, ec.Collection, path+".collection", file)

	for j, then := range ec.Then {
		findings = checkEnsuresConfigRefs(findings, st, then,
			fmt.Sprintf("%s.then[%d]", path, j), file)
	}
	for j, el := range ec.Else {
		findings = checkEnsuresConfigRefs(findings, st, el,
			fmt.Sprintf("%s.else[%d]", path, j), file)
	}
	for j, body := range ec.Body {
		findings = checkEnsuresConfigRefs(findings, st, body,
			fmt.Sprintf("%s.body[%d]", path, j), file)
	}

	return findings
}

// checkProvidesItemTrigger checks that a provides item's trigger resolves (RULE-30).
// Recurses into for_each items.
func checkProvidesItemTrigger(findings []report.Finding, spec *ast.Spec, st *SymbolTable, p ast.ProvidesItem, surfaceName string, path string) []report.Finding {
	switch p.Kind {
	case "action":
		if p.Trigger != "" {
			triggers := st.LookupTrigger(p.Trigger)
			if len(triggers) == 0 {
				findings = append(findings, report.NewError(
					"RULE-30",
					fmt.Sprintf("Surface '%s' provides trigger '%s' not declared in any rule", surfaceName, p.Trigger),
					report.Location{File: spec.File, Path: path + ".trigger"},
				))
			}
		}
	case "for_each":
		for j, item := range p.Items {
			findings = checkProvidesItemTrigger(findings, spec, st, item, surfaceName,
				fmt.Sprintf("%s.items[%d]", path, j))
		}
	}
	return findings
}
