package semantic

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/foundry-zero/allium/internal/ast"
	"github.com/foundry-zero/allium/internal/report"
)

// CheckStateMachines analyzes entity lifecycle state machines.
//
//   - RULE-07: All status enum values must be reachable from creation points via BFS
//   - RULE-08: Non-terminal status values must have at least one outgoing transition
//   - RULE-09: Ensures clauses must only assign values declared in the enum
func CheckStateMachines(spec *ast.Spec, st *SymbolTable) []report.Finding {
	var findings []report.Finding

	for i, entity := range spec.Entities {
		enumField, enumValues := findStatusEnum(entity, st)
		if enumField == "" {
			continue
		}

		valueSet := make(map[string]bool, len(enumValues))
		for _, v := range enumValues {
			valueSet[v] = true
		}

		// Collect creation values and transitions from rules
		creationValues, transitions, undeclared := collectStateInfo(spec, st, entity.Name, enumField, valueSet)

		// RULE-09: Report assignments to undeclared enum values
		for _, u := range undeclared {
			findings = append(findings, report.NewError(
				"RULE-09",
				fmt.Sprintf("Undeclared status value '%s' assigned to '%s.%s'", u.value, entity.Name, enumField),
				report.Location{File: spec.File, Path: u.path},
			))
		}

		// RULE-07: BFS reachability from creation values
		reachable := bfsReachable(creationValues, transitions)
		for _, v := range enumValues {
			if !reachable[v] {
				findings = append(findings, report.NewError(
					"RULE-07",
					fmt.Sprintf("Unreachable status value '%s' on '%s'", v, entity.Name),
					report.Location{File: spec.File, Path: fmt.Sprintf("$.entities[%d]", i)},
				))
			}
		}

		// RULE-08: Non-terminal values must have outgoing transitions
		// Terminal values are those with no outgoing transitions that ARE reachable
		// We only flag reachable values with no outgoing edges
		outgoing := make(map[string]bool)
		for from := range transitions {
			outgoing[from] = true
		}
		for _, v := range enumValues {
			if reachable[v] && !outgoing[v] && !isInCreationValues(v, creationValues) {
				// Value is reachable but has no way out — could be terminal or dead-end
				// We report it as RULE-08 (dead-end) since truly terminal states
				// are intentional and rare; the spec author can suppress if intended
				findings = append(findings, report.NewError(
					"RULE-08",
					fmt.Sprintf("Dead-end state '%s' on '%s' has no outgoing transition", v, entity.Name),
					report.Location{File: spec.File, Path: fmt.Sprintf("$.entities[%d]", i)},
				))
			}
		}
	}

	return findings
}

// findStatusEnum finds the first enum-typed field on an entity (typically named "status").
// Returns the field name and enum values, or empty if none found.
func findStatusEnum(entity ast.Entity, st *SymbolTable) (string, []string) {
	for _, f := range entity.Fields {
		switch f.Type.Kind {
		case "named_enum":
			if enum := st.LookupEnumeration(f.Type.Name); enum != nil {
				return f.Name, enum.Values
			}
		case "inline_enum":
			return f.Name, f.Type.Values
		}
	}
	return "", nil
}

type undeclaredAssignment struct {
	value string
	path  string
}

// collectStateInfo scans all rules for creation values and transitions
// for the given entity's enum field.
func collectStateInfo(spec *ast.Spec, _ *SymbolTable, entityName string, enumField string, validValues map[string]bool) (
	creationValues []string,
	transitions map[string][]string,
	undeclared []undeclaredAssignment,
) {
	transitions = make(map[string][]string)

	for i, rule := range spec.Rules {
		basePath := fmt.Sprintf("$.rules[%d]", i)

		triggerEntity := rule.Trigger.Entity

		// Build a set of binding names that resolve to the target entity.
		// This prevents false matches when two entities share a field name.
		entityBindings := make(map[string]bool)
		if rule.Trigger.Binding != "" && triggerEntity == entityName {
			entityBindings[rule.Trigger.Binding] = true
		}
		// Also track let_bindings that do join_lookup on our entity
		for _, lb := range rule.LetBindings {
			if lb.Expression != nil && lb.Expression.Kind == "join_lookup" && lb.Expression.Entity == entityName {
				entityBindings[lb.Name] = true
			}
		}

		for j, ec := range rule.Ensures {
			ecPath := fmt.Sprintf("%s.ensures[%d]", basePath, j)
			creationValues, transitions, undeclared = collectEnsuresStateInfo(
				ec, ecPath, entityName, enumField, triggerEntity, entityBindings, validValues,
				creationValues, transitions, undeclared,
			)
		}
	}

	return
}

// collectEnsuresStateInfo recursively processes ensures clauses.
func collectEnsuresStateInfo(
	ec ast.EnsuresClause,
	path string,
	entityName string,
	enumField string,
	triggerEntity string,
	entityBindings map[string]bool,
	validValues map[string]bool,
	creationValues []string,
	transitions map[string][]string,
	undeclared []undeclaredAssignment,
) ([]string, map[string][]string, []undeclaredAssignment) {

	switch ec.Kind {
	case "entity_creation":
		if ec.Entity == entityName {
			// Look for the enum field in creation fields
			if ec.Fields != nil {
				if fieldExpr, ok := ec.Fields[enumField]; ok {
					val := extractLiteralValue(&fieldExpr)
					if val != "" {
						creationValues = append(creationValues, val)
						if !validValues[val] {
							undeclared = append(undeclared, undeclaredAssignment{
								value: val,
								path:  fmt.Sprintf("%s.fields.%s", path, enumField),
							})
						}
					}
				}
			}
		}

	case "state_change":
		// Strict match: entity context confirmed — used for RULE-09 error reporting.
		// Loose match: field name matches but entity unknown — used for transition tracking only.
		strictMatch := isFieldAccessFor(ec.Target, enumField, triggerEntity, entityName, entityBindings)
		looseMatch := strictMatch || isFieldAccessForFieldOnly(ec.Target, enumField)

		if looseMatch {
			newVal := extractRawValue(ec.Value)
			if newVal != "" {
				// Only report RULE-09 when entity context is confirmed
				if strictMatch && !validValues[newVal] {
					undeclared = append(undeclared, undeclaredAssignment{
						value: newVal,
						path:  path + ".value",
					})
				}

				// Track transitions for RULE-07/08 regardless of strict/loose
				fromVal := extractFromState(ec)
				if fromVal != "" {
					transitions[fromVal] = append(transitions[fromVal], newVal)
				} else {
					for v := range validValues {
						if v != newVal {
							transitions[v] = append(transitions[v], newVal)
						}
					}
				}
			}
		}

	case "conditional":
		for i, then := range ec.Then {
			creationValues, transitions, undeclared = collectEnsuresStateInfo(
				then, fmt.Sprintf("%s.then[%d]", path, i),
				entityName, enumField, triggerEntity, entityBindings, validValues,
				creationValues, transitions, undeclared,
			)
		}
		for i, el := range ec.Else {
			creationValues, transitions, undeclared = collectEnsuresStateInfo(
				el, fmt.Sprintf("%s.else[%d]", path, i),
				entityName, enumField, triggerEntity, entityBindings, validValues,
				creationValues, transitions, undeclared,
			)
		}

	case "iteration":
		for i, body := range ec.Body {
			creationValues, transitions, undeclared = collectEnsuresStateInfo(
				body, fmt.Sprintf("%s.body[%d]", path, i),
				entityName, enumField, triggerEntity, entityBindings, validValues,
				creationValues, transitions, undeclared,
			)
		}

	case "let_binding":
		// Handle entity_creation nested inside ensures let_binding.
		// e.g., {kind: "let_binding", name: "token", value: {kind: "entity_creation", ...}}
		if ec.Value != nil {
			var innerEC ast.EnsuresClause
			if err := json.Unmarshal(ec.Value, &innerEC); err == nil && innerEC.Kind != "" {
				creationValues, transitions, undeclared = collectEnsuresStateInfo(
					innerEC, path+".value",
					entityName, enumField, triggerEntity, entityBindings, validValues,
					creationValues, transitions, undeclared,
				)
			}
		}
		for i, body := range ec.Body {
			creationValues, transitions, undeclared = collectEnsuresStateInfo(
				body, fmt.Sprintf("%s.body[%d]", path, i),
				entityName, enumField, triggerEntity, entityBindings, validValues,
				creationValues, transitions, undeclared,
			)
		}
	}

	return creationValues, transitions, undeclared
}

// isFieldAccessFor checks if an expression is a field access for the given field name
// on the correct entity. It verifies entity context to prevent false matches when
// two entities share a field name (e.g., User.status vs Session.status).
func isFieldAccessFor(expr *ast.Expression, fieldName string, triggerEntity string, entityName string, entityBindings map[string]bool) bool {
	if expr == nil || expr.Kind != "field_access" {
		return false
	}
	if expr.Field != fieldName {
		return false
	}

	// Root access (e.g., just "status") — the field is on the trigger entity
	if expr.Object == nil {
		return triggerEntity == entityName
	}

	// Chained access (e.g., "session.status") — check if the root binding
	// resolves to the target entity
	if expr.Object.Kind == "field_access" && expr.Object.Object == nil {
		return entityBindings[expr.Object.Field]
	}

	// Deeper chains — conservatively skip (don't match)
	return false
}

// isFieldAccessForFieldOnly checks if an expression accesses the given field name,
// without verifying entity context. Used for loose transition tracking when entity
// bindings can't be resolved (e.g., external_stimulus trigger parameters).
func isFieldAccessForFieldOnly(expr *ast.Expression, fieldName string) bool {
	if expr == nil || expr.Kind != "field_access" {
		return false
	}
	return expr.Field == fieldName
}

// extractLiteralValue extracts a string value from a literal expression.
func extractLiteralValue(expr *ast.Expression) string {
	if expr == nil {
		return ""
	}
	if expr.Kind == "literal" {
		var s string
		if err := json.Unmarshal(expr.LitValue, &s); err == nil {
			return s
		}
	}
	return ""
}

// extractRawValue extracts a string value from a raw JSON message (for ensures value).
func extractRawValue(raw json.RawMessage) string {
	if raw == nil {
		return ""
	}

	// Try to unmarshal as an Expression first
	var expr ast.Expression
	if err := json.Unmarshal(raw, &expr); err == nil {
		return extractLiteralValue(&expr)
	}

	// Try as a plain string
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}

	return ""
}

// extractFromState tries to determine the "from" state from a state_change ensures.
// This is heuristic — in practice, the trigger often constrains the from-state.
func extractFromState(_ ast.EnsuresClause) string {
	// State transitions are typically guarded by trigger conditions,
	// not explicitly encoded in the ensures clause.
	// Return empty to use the conservative all-to-all approach.
	return ""
}

// bfsReachable performs BFS from creation values through transitions.
func bfsReachable(seeds []string, transitions map[string][]string) map[string]bool {
	reachable := make(map[string]bool)
	queue := make([]string, 0, len(seeds))

	for _, s := range seeds {
		if !reachable[s] {
			reachable[s] = true
			queue = append(queue, s)
		}
	}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		for _, next := range transitions[curr] {
			if !reachable[next] {
				reachable[next] = true
				queue = append(queue, next)
			}
		}
	}

	return reachable
}

// isInCreationValues checks if a value is one of the creation seeds.
func isInCreationValues(v string, creationValues []string) bool {
	return slices.Contains(creationValues, v)
}
