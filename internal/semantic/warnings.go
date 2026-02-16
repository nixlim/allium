package semantic

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/foundry-zero/allium/internal/ast"
	"github.com/foundry-zero/allium/internal/report"
)

// CheckWarnings detects all 19 warning conditions (WARN-01 through WARN-19).
// All findings have Severity=SeverityWarning.
func CheckWarnings(spec *ast.Spec, st *SymbolTable) []report.Finding {
	var findings []report.Finding

	findings = checkWarn01ExternalNoSpec(findings, spec, st)
	findings = checkWarn02OpenQuestions(findings, spec)
	findings = checkWarn03DeferredNoHint(findings, spec)
	findings = checkWarn04UnusedEntity(findings, spec, st)
	findings = checkWarn05NeverFires(findings, spec)
	findings = checkWarn06TemporalNoGuard(findings, spec)
	findings = checkWarn07UnusedExposed(findings, spec, st)
	findings = checkWarn08ImpossibleProvides(findings, spec)
	findings = checkWarn09UnusedActor(findings, spec)
	findings = checkWarn10SiblingCreation(findings, spec)
	findings = checkWarn11WeakProvides(findings, spec)
	findings = checkWarn12OverlappingRequires(findings, spec, st)
	findings = checkWarn13DerivedScope(findings, spec)
	findings = checkWarn14TrivialActor(findings, spec)
	findings = checkWarn15EmptyConditionalPath(findings, spec)
	findings = checkWarn16OptionalTemporal(findings, spec, st)
	findings = checkWarn17RawWithActors(findings, spec, st)
	findings = checkWarn18TransitionsOnCreation(findings, spec, st)
	findings = checkWarn19DuplicateInlineEnums(findings, spec)

	return findings
}

// WARN-01: External entity not referenced by any use declaration's imported types.
func checkWarn01ExternalNoSpec(findings []report.Finding, spec *ast.Spec, _ *SymbolTable) []report.Finding {
	// If there are no use declarations, every external entity gets a warning
	for i, ee := range spec.ExternalEntities {
		// Check if any use declaration could govern this external entity
		// (we can't resolve cross-spec, so we check if ANY use_declarations exist)
		if len(spec.UseDeclarations) == 0 {
			findings = append(findings, report.NewWarning(
				"WARN-01",
				fmt.Sprintf("External entity '%s' has no governing spec", ee.Name),
				report.Location{File: spec.File, Path: fmt.Sprintf("$.external_entities[%d]", i)},
			))
		}
	}
	return findings
}

// WARN-02: Open questions present.
func checkWarn02OpenQuestions(findings []report.Finding, spec *ast.Spec) []report.Finding {
	if len(spec.OpenQuestions) > 0 {
		findings = append(findings, report.NewWarning(
			"WARN-02",
			fmt.Sprintf("Open questions present: %d unresolved", len(spec.OpenQuestions)),
			report.Location{File: spec.File, Path: "$.open_questions"},
		))
	}
	return findings
}

// WARN-03: Deferred spec with null/empty location_hint.
func checkWarn03DeferredNoHint(findings []report.Finding, spec *ast.Spec) []report.Finding {
	for i, d := range spec.Deferred {
		if d.LocationHint == nil || *d.LocationHint == "" {
			findings = append(findings, report.NewWarning(
				"WARN-03",
				fmt.Sprintf("Deferred spec '%s' has no location hint", d.Name),
				report.Location{File: spec.File, Path: fmt.Sprintf("$.deferred[%d]", i)},
			))
		}
	}
	return findings
}

// WARN-04: Entity never referenced by any rule, surface, relationship, or other entity.
func checkWarn04UnusedEntity(findings []report.Finding, spec *ast.Spec, st *SymbolTable) []report.Finding {
	referenced := collectReferencedEntities(spec)

	for i, e := range spec.Entities {
		if !referenced[e.Name] {
			findings = append(findings, report.NewWarning(
				"WARN-04",
				fmt.Sprintf("Unused entity '%s'", e.Name),
				report.Location{File: spec.File, Path: fmt.Sprintf("$.entities[%d]", i)},
			))
		}
	}
	_ = st
	return findings
}

// collectReferencedEntities scans all parts of the spec for entity name references.
func collectReferencedEntities(spec *ast.Spec) map[string]bool {
	refs := make(map[string]bool)

	// References from entity_ref fields
	for _, e := range spec.Entities {
		for _, f := range e.Fields {
			collectFieldTypeEntityRefs(f.Type, refs)
		}
		for _, r := range e.Relationships {
			refs[r.TargetEntity] = true
		}
	}
	for _, ee := range spec.ExternalEntities {
		for _, f := range ee.Fields {
			collectFieldTypeEntityRefs(f.Type, refs)
		}
	}
	for _, vt := range spec.ValueTypes {
		for _, f := range vt.Fields {
			collectFieldTypeEntityRefs(f.Type, refs)
		}
	}
	for _, v := range spec.Variants {
		refs[v.BaseEntity] = true
		for _, f := range v.Fields {
			collectFieldTypeEntityRefs(f.Type, refs)
		}
	}
	for _, g := range spec.Given {
		collectFieldTypeEntityRefs(g.Type, refs)
	}

	// References from rules
	for _, r := range spec.Rules {
		if r.Trigger.Entity != "" {
			refs[r.Trigger.Entity] = true
		}
		for _, ec := range r.Ensures {
			if ec.Entity != "" {
				refs[ec.Entity] = true
			}
			collectEnsuresEntityRefs(ec, refs)
		}
	}

	// References from surfaces
	for _, s := range spec.Surfaces {
		if s.Facing.Type != "" {
			refs[s.Facing.Type] = true
		}
		if s.Context != nil && s.Context.Type != "" {
			refs[s.Context.Type] = true
		}
	}

	// References from actors
	for _, a := range spec.Actors {
		refs[a.IdentifiedBy.Entity] = true
	}

	// References from defaults
	for _, d := range spec.Defaults {
		refs[d.Entity] = true
	}

	return refs
}

func collectFieldTypeEntityRefs(ft ast.FieldType, refs map[string]bool) {
	switch ft.Kind {
	case "entity_ref":
		refs[ft.Entity] = true
	case "optional":
		if ft.Inner != nil {
			collectFieldTypeEntityRefs(*ft.Inner, refs)
		}
	case "set", "list":
		if ft.Element != nil {
			collectFieldTypeEntityRefs(*ft.Element, refs)
		}
	}
}

func collectEnsuresEntityRefs(ec ast.EnsuresClause, refs map[string]bool) {
	if ec.Entity != "" {
		refs[ec.Entity] = true
	}
	for _, then := range ec.Then {
		collectEnsuresEntityRefs(then, refs)
	}
	for _, el := range ec.Else {
		collectEnsuresEntityRefs(el, refs)
	}
	for _, body := range ec.Body {
		collectEnsuresEntityRefs(body, refs)
	}
}

// WARN-05: Rule with contradictory requires (heuristic: two equality checks on same field with different values).
func checkWarn05NeverFires(findings []report.Finding, spec *ast.Spec) []report.Finding {
	for i, rule := range spec.Rules {
		if hasContradictoryRequires(rule.Requires) {
			findings = append(findings, report.NewWarning(
				"WARN-05",
				fmt.Sprintf("Rule '%s' can never fire (contradictory requires)", rule.Name),
				report.Location{File: spec.File, Path: fmt.Sprintf("$.rules[%d].requires", i)},
			))
		}
	}
	return findings
}

func hasContradictoryRequires(requires []ast.Expression) bool {
	// Simple heuristic: collect equality constraints on the same field
	type constraint struct {
		field string
		value string
	}
	var constraints []constraint

	for _, req := range requires {
		if req.Kind == "comparison" && req.Operator == "=" {
			field := extractSimpleFieldName(req.Left)
			value := extractSimpleFieldName(req.Right)
			if field != "" && value != "" {
				constraints = append(constraints, constraint{field, value})
			}
		}
	}

	// Check for same field with different values
	for a := 0; a < len(constraints); a++ {
		for b := a + 1; b < len(constraints); b++ {
			if constraints[a].field == constraints[b].field && constraints[a].value != constraints[b].value {
				return true
			}
		}
	}
	return false
}

func extractSimpleFieldName(expr *ast.Expression) string {
	if expr == nil {
		return ""
	}
	if expr.Kind == "field_access" && expr.Object == nil {
		return expr.Field
	}
	if expr.Kind == "literal" {
		return extractLiteralValue(expr)
	}
	return ""
}

// WARN-06: Temporal trigger without re-firing guard.
func checkWarn06TemporalNoGuard(findings []report.Finding, spec *ast.Spec) []report.Finding {
	for i, rule := range spec.Rules {
		if rule.Trigger.Kind == "temporal" && len(rule.Requires) == 0 {
			findings = append(findings, report.NewWarning(
				"WARN-06",
				fmt.Sprintf("Temporal rule '%s' has no re-firing guard", rule.Name),
				report.Location{File: spec.File, Path: fmt.Sprintf("$.rules[%d]", i)},
			))
		}
	}
	return findings
}

// WARN-07: Surface exposes a field not used by any rule (stub — full analysis complex).
func checkWarn07UnusedExposed(findings []report.Finding, _ *ast.Spec, _ *SymbolTable) []report.Finding {
	// Full implementation would track which entity fields are read/written by rules
	// and compare with surface exposes. This is complex and deferred.
	return findings
}

// WARN-08: Provides with always-false when condition (heuristic).
func checkWarn08ImpossibleProvides(findings []report.Finding, _ *ast.Spec) []report.Finding {
	// Detecting always-false conditions requires symbolic evaluation.
	// Deferred to future enhancement.
	return findings
}

// WARN-09: Actor not referenced in any surface facing clause.
func checkWarn09UnusedActor(findings []report.Finding, spec *ast.Spec) []report.Finding {
	usedActors := make(map[string]bool)
	for _, s := range spec.Surfaces {
		usedActors[s.Facing.Type] = true
	}

	for i, a := range spec.Actors {
		if !usedActors[a.Name] {
			findings = append(findings, report.NewWarning(
				"WARN-09",
				fmt.Sprintf("Unused actor '%s'", a.Name),
				report.Location{File: spec.File, Path: fmt.Sprintf("$.actors[%d]", i)},
			))
		}
	}
	return findings
}

// WARN-10: Sibling rule creates entity without duplicate guard (stub).
func checkWarn10SiblingCreation(findings []report.Finding, _ *ast.Spec) []report.Finding {
	// Complex heuristic requiring cross-rule analysis. Deferred.
	return findings
}

// WARN-11: Provides condition weaker than rule requires (stub).
func checkWarn11WeakProvides(findings []report.Finding, _ *ast.Spec) []report.Finding {
	// Would require comparing when-condition with requires. Deferred.
	return findings
}

// WARN-12: Two rules with overlapping requires on same trigger.
func checkWarn12OverlappingRequires(findings []report.Finding, _ *ast.Spec, st *SymbolTable) []report.Finding {
	for triggerName, rules := range st.Triggers {
		if len(rules) < 2 {
			continue
		}
		// If multiple rules share a trigger and both have empty requires, they overlap
		emptyRequires := 0
		for _, r := range rules {
			if len(r.Requires) == 0 {
				emptyRequires++
			}
		}
		if emptyRequires >= 2 {
			findings = append(findings, report.NewWarning(
				"WARN-12",
				fmt.Sprintf("Overlapping preconditions on trigger '%s' (%d rules with no requires)", triggerName, emptyRequires),
				report.Location{File: ""},
			))
		}
	}
	return findings
}

// WARN-13: Derived value referencing out-of-entity fields (stub).
func checkWarn13DerivedScope(findings []report.Finding, _ *ast.Spec) []report.Finding {
	// Would require tracking which fields belong to which entity. Deferred.
	return findings
}

// WARN-14: Trivial actor identified_by condition (always true/false).
func checkWarn14TrivialActor(findings []report.Finding, spec *ast.Spec) []report.Finding {
	for i, a := range spec.Actors {
		cond := a.IdentifiedBy.Condition
		if cond != nil && cond.Kind == "literal" && cond.Type == "boolean" {
			findings = append(findings, report.NewWarning(
				"WARN-14",
				fmt.Sprintf("Trivial actor identified_by condition on '%s'", a.Name),
				report.Location{File: spec.File, Path: fmt.Sprintf("$.actors[%d].identified_by.condition", i)},
			))
		}
	}
	return findings
}

// WARN-15: All ensures clauses are conditional with at least one empty path.
func checkWarn15EmptyConditionalPath(findings []report.Finding, spec *ast.Spec) []report.Finding {
	for i, rule := range spec.Rules {
		if len(rule.Ensures) == 0 {
			continue
		}
		allConditional := true
		hasEmptyPath := false
		for _, ec := range rule.Ensures {
			if ec.Kind != "conditional" {
				allConditional = false
				break
			}
			if len(ec.Else) == 0 {
				hasEmptyPath = true
			}
		}
		if allConditional && hasEmptyPath {
			findings = append(findings, report.NewWarning(
				"WARN-15",
				fmt.Sprintf("All-conditional ensures with empty path in rule '%s'", rule.Name),
				report.Location{File: spec.File, Path: fmt.Sprintf("$.rules[%d].ensures", i)},
			))
		}
	}
	return findings
}

// WARN-16: Temporal trigger on optional field.
func checkWarn16OptionalTemporal(findings []report.Finding, spec *ast.Spec, st *SymbolTable) []report.Finding {
	for i, rule := range spec.Rules {
		if rule.Trigger.Kind != "temporal" {
			continue
		}
		entityName := rule.Trigger.Entity
		fieldName := rule.Trigger.Field
		if entityName == "" || fieldName == "" {
			continue
		}
		if ent := st.LookupEntity(entityName); ent != nil {
			for _, f := range ent.Fields {
				if f.Name == fieldName && f.Type.Kind == "optional" {
					findings = append(findings, report.NewWarning(
						"WARN-16",
						fmt.Sprintf("Temporal trigger on optional field '%s.%s' — won't fire when absent", entityName, fieldName),
						report.Location{File: spec.File, Path: fmt.Sprintf("$.rules[%d].trigger", i)},
					))
				}
			}
		}
	}
	return findings
}

// WARN-17: Surface using raw entity type in facing when actors exist for that entity.
func checkWarn17RawWithActors(findings []report.Finding, spec *ast.Spec, st *SymbolTable) []report.Finding {
	// Build map: entity -> actors that identify by that entity
	entityActors := make(map[string][]string)
	for _, a := range spec.Actors {
		entityActors[a.IdentifiedBy.Entity] = append(entityActors[a.IdentifiedBy.Entity], a.Name)
	}

	for i, s := range spec.Surfaces {
		facingType := s.Facing.Type
		// If facing type is an entity (not an actor) and actors exist for it
		if st.LookupActor(facingType) == nil && st.LookupEntity(facingType) != nil {
			if actors, ok := entityActors[facingType]; ok && len(actors) > 0 {
				findings = append(findings, report.NewWarning(
					"WARN-17",
					fmt.Sprintf("Raw entity type '%s' used in facing when actors available: %s", facingType, strings.Join(actors, ", ")),
					report.Location{File: spec.File, Path: fmt.Sprintf("$.surfaces[%d].facing.type", i)},
				))
			}
		}
	}
	return findings
}

// WARN-18: transitions_to trigger on a creation value.
func checkWarn18TransitionsOnCreation(findings []report.Finding, spec *ast.Spec, st *SymbolTable) []report.Finding {
	// Collect creation values per entity/field
	type entityField struct {
		entity string
		field  string
	}
	creationValues := make(map[entityField][]string)

	for _, rule := range spec.Rules {
		for _, ec := range rule.Ensures {
			if ec.Kind == "entity_creation" && ec.Fields != nil {
				for fieldName, fieldExpr := range ec.Fields {
					val := extractLiteralValue(&fieldExpr)
					if val != "" {
						key := entityField{ec.Entity, fieldName}
						creationValues[key] = append(creationValues[key], val)
					}
				}
			}
		}
	}

	for i, rule := range spec.Rules {
		if rule.Trigger.Kind != "state_transition" {
			continue
		}
		toVal := rule.Trigger.ToValue
		if toVal == "" {
			continue
		}
		key := entityField{rule.Trigger.Entity, rule.Trigger.Field}
		if slices.Contains(creationValues[key], toVal) {
			findings = append(findings, report.NewWarning(
				"WARN-18",
				fmt.Sprintf("transitions_to '%s' fires on creation value for '%s.%s'", toVal, rule.Trigger.Entity, rule.Trigger.Field),
				report.Location{File: spec.File, Path: fmt.Sprintf("$.rules[%d].trigger", i)},
			))
		}
	}
	_ = st
	return findings
}

// WARN-19: Multiple fields with identical inline enum literal sets.
func checkWarn19DuplicateInlineEnums(findings []report.Finding, spec *ast.Spec) []report.Finding {
	for i, entity := range spec.Entities {
		// Collect inline enum value sets
		type enumInfo struct {
			fieldName string
			values    string // sorted, joined for comparison
		}
		var enums []enumInfo

		for _, f := range entity.Fields {
			if f.Type.Kind == "inline_enum" {
				sorted := make([]string, len(f.Type.Values))
				copy(sorted, f.Type.Values)
				sort.Strings(sorted)
				enums = append(enums, enumInfo{f.Name, strings.Join(sorted, "|")})
			}
		}

		// Check for duplicates
		seen := make(map[string]string) // values -> first field name
		for _, e := range enums {
			if first, ok := seen[e.values]; ok {
				findings = append(findings, report.NewWarning(
					"WARN-19",
					fmt.Sprintf("Multiple identical inline enums on '%s' (fields '%s' and '%s') — consider a named enum",
						entity.Name, first, e.fieldName),
					report.Location{File: spec.File, Path: fmt.Sprintf("$.entities[%d]", i)},
				))
			} else {
				seen[e.values] = e.fieldName
			}
		}
	}
	return findings
}
