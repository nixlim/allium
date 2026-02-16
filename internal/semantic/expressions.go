package semantic

import (
	"encoding/json"
	"fmt"
	"maps"

	"github.com/foundry-zero/allium/internal/ast"
	"github.com/foundry-zero/allium/internal/report"
)

// CheckExpressions validates expression correctness.
//
//   - RULE-10: Derived value dependency cycles (Tarjan SCC)
//   - RULE-11: All field_access roots must be in scope
//   - RULE-12: Type compatibility in comparisons and arithmetic
//   - RULE-13: any/all expressions must have explicit lambda parameters
//   - RULE-14: Inline enum comparisons are forbidden; named enum comparisons must be same type
func CheckExpressions(spec *ast.Spec, st *SymbolTable) []report.Finding {
	var findings []report.Finding

	// RULE-10: Derived value cycle detection
	findings = checkDerivedValueCycles(findings, spec)

	// RULE-11: Out-of-scope field access in rules
	findings = checkRuleScopes(findings, spec, st)

	// RULE-12: Type mismatches in comparisons and arithmetic
	findings = checkTypeMismatches(findings, spec, st)

	// RULE-13: any/all lambda parameter check
	findings = checkCollectionOps(findings, spec)

	// RULE-14: Enum comparison check
	findings = checkEnumComparisons(findings, spec, st)

	return findings
}

// --- RULE-10: Derived value cycle detection using Tarjan's SCC ---

func checkDerivedValueCycles(findings []report.Finding, spec *ast.Spec) []report.Finding {
	// Check entity derived values
	for i, entity := range spec.Entities {
		if len(entity.DerivedValues) < 2 {
			continue
		}
		findings = detectDerivedCycles(findings, entity.DerivedValues,
			fmt.Sprintf("$.entities[%d].derived_values", i), spec.File)
	}

	// Check value type derived values
	for i, vt := range spec.ValueTypes {
		if len(vt.DerivedValues) < 2 {
			continue
		}
		findings = detectDerivedCycles(findings, vt.DerivedValues,
			fmt.Sprintf("$.value_types[%d].derived_values", i), spec.File)
	}

	return findings
}

// detectDerivedCycles runs Tarjan's SCC on the derived value dependency graph
// and reports any multi-node strongly connected components (cycles).
func detectDerivedCycles(findings []report.Finding, dvs []ast.DerivedValue, path string, file string) []report.Finding {
	// Build name -> index and adjacency list
	nameIdx := make(map[string]int, len(dvs))
	for j, dv := range dvs {
		nameIdx[dv.Name] = j
	}

	adj := make([][]int, len(dvs))
	for j, dv := range dvs {
		adj[j] = collectDerivedRefs(dv.Expression, nameIdx)
	}

	// Run Tarjan's SCC
	sccs := tarjanSCC(adj)
	for _, scc := range sccs {
		if len(scc) > 1 {
			names := make([]string, len(scc))
			for k, idx := range scc {
				names[k] = dvs[idx].Name
			}
			// Add first name again to close the cycle in the message
			names = append(names, names[0])
			findings = append(findings, report.NewError(
				"RULE-10",
				fmt.Sprintf("Cycle detected in derived values: %s", joinArrow(names)),
				report.Location{File: file, Path: path},
			))
		}
	}

	return findings
}

// collectDerivedRefs finds which derived values an expression references.
func collectDerivedRefs(expr *ast.Expression, nameIdx map[string]int) []int {
	if expr == nil {
		return nil
	}
	var refs []int
	seen := make(map[int]bool)

	var walk func(e *ast.Expression)
	walk = func(e *ast.Expression) {
		if e == nil {
			return
		}
		if e.Kind == "field_access" && e.Object == nil {
			if idx, ok := nameIdx[e.Field]; ok && !seen[idx] {
				refs = append(refs, idx)
				seen[idx] = true
			}
		}
		walk(e.Object)
		walk(e.Left)
		walk(e.Right)
		walk(e.Target)
		walk(e.Operand)
		walk(e.Collection)
		walk(e.Lambda)
		walk(e.Condition)
		walk(e.Body)
		walk(e.Element)
		for j := range e.FuncArguments {
			walk(&e.FuncArguments[j])
		}
		for j := range e.Elements {
			walk(&e.Elements[j])
		}
	}
	walk(expr)
	return refs
}

// tarjanSCC returns strongly connected components using Tarjan's algorithm.
func tarjanSCC(adj [][]int) [][]int {
	n := len(adj)
	index := make([]int, n)
	lowlink := make([]int, n)
	onStack := make([]bool, n)
	defined := make([]bool, n)
	stack := make([]int, 0, n)
	var sccs [][]int
	counter := 0

	var strongConnect func(v int)
	strongConnect = func(v int) {
		index[v] = counter
		lowlink[v] = counter
		counter++
		defined[v] = true
		stack = append(stack, v)
		onStack[v] = true

		for _, w := range adj[v] {
			if !defined[w] {
				strongConnect(w)
				lowlink[v] = min(lowlink[v], lowlink[w])
			} else if onStack[w] {
				lowlink[v] = min(lowlink[v], index[w])
			}
		}

		if lowlink[v] == index[v] {
			var scc []int
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				scc = append(scc, w)
				if w == v {
					break
				}
			}
			sccs = append(sccs, scc)
		}
	}

	for v := range n {
		if !defined[v] {
			strongConnect(v)
		}
	}

	return sccs
}

func joinArrow(parts []string) string {
	result := parts[0]
	for _, p := range parts[1:] {
		result += " -> " + p
	}
	return result
}

// --- RULE-11: Out-of-scope field access ---

// checkRuleScopes validates that every root field_access in a rule's expressions
// references an identifier that is in scope for that rule.
func checkRuleScopes(findings []report.Finding, spec *ast.Spec, _ *SymbolTable) []report.Finding {
	// Build global scope identifiers (given bindings, config params, default instance names)
	globalScope := make(map[string]bool)
	for _, g := range spec.Given {
		globalScope[g.Name] = true
	}
	for _, c := range spec.Config {
		globalScope[c.Name] = true
	}
	for _, d := range spec.Defaults {
		globalScope[d.Name] = true
	}
	// "config" is an implicit root that provides access to config params
	globalScope["config"] = true

	for i, rule := range spec.Rules {
		basePath := fmt.Sprintf("$.rules[%d]", i)
		scope := buildRuleScope(rule, globalScope)

		// Check let bindings (each let binding expression can reference earlier names)
		letScope := copyScope(scope)
		for j, lb := range rule.LetBindings {
			findings = walkForScopeViolations(findings, lb.Expression, letScope,
				fmt.Sprintf("%s.let_bindings[%d].expression", basePath, j), spec.File)
			// Add the let binding name to scope for subsequent bindings
			letScope[lb.Name] = true
		}

		// Build full scope including all let bindings for requires and ensures
		fullScope := copyScope(scope)
		for _, lb := range rule.LetBindings {
			fullScope[lb.Name] = true
		}

		// Check requires (let bindings are in scope for requires)
		for j, req := range rule.Requires {
			findings = walkForScopeViolations(findings, &req, fullScope,
				fmt.Sprintf("%s.requires[%d]", basePath, j), spec.File)
		}

		// Check for_clause
		if rule.ForClause != nil {
			findings = walkForScopeViolations(findings, rule.ForClause.Collection, fullScope,
				fmt.Sprintf("%s.for_clause.collection", basePath), spec.File)
			if rule.ForClause.Condition != nil {
				forScope := copyScope(fullScope)
				forScope[rule.ForClause.Binding] = true
				findings = walkForScopeViolations(findings, rule.ForClause.Condition, forScope,
					fmt.Sprintf("%s.for_clause.condition", basePath), spec.File)
			}
			// Add for-clause binding to scope for ensures
			fullScope[rule.ForClause.Binding] = true
		}

		// Check ensures clauses
		for j, ec := range rule.Ensures {
			findings = walkEnsuresForScopeViolations(findings, ec, fullScope,
				fmt.Sprintf("%s.ensures[%d]", basePath, j), spec.File)
		}
	}

	return findings
}

// buildRuleScope constructs the set of identifiers in scope for a rule.
func buildRuleScope(rule ast.Rule, globalScope map[string]bool) map[string]bool {
	scope := copyScope(globalScope)

	// Trigger binding name (for binding-type triggers)
	if rule.Trigger.Binding != "" {
		scope[rule.Trigger.Binding] = true
	}

	// Trigger parameters (for external_stimulus and chained triggers)
	for _, p := range rule.Trigger.Parameters {
		scope[p.Name] = true
	}

	return scope
}

// copyScope creates a shallow copy of a scope map.
func copyScope(src map[string]bool) map[string]bool {
	dst := make(map[string]bool, len(src))
	maps.Copy(dst, src)
	return dst
}

// walkForScopeViolations walks an expression tree and reports root field_access
// identifiers that are not in the given scope.
func walkForScopeViolations(findings []report.Finding, expr *ast.Expression, scope map[string]bool, path string, file string) []report.Finding {
	if expr == nil {
		return findings
	}

	if expr.Kind == "field_access" && expr.Object == nil {
		if !scope[expr.Field] {
			findings = append(findings, report.NewError(
				"RULE-11",
				fmt.Sprintf("Identifier '%s' is not in scope", expr.Field),
				report.Location{File: file, Path: path},
			))
		}
		return findings // no need to recurse into a root field_access
	}

	// For lambda expressions, add parameter to scope for the body
	if expr.Kind == "lambda" && expr.Parameter != "" {
		lambdaScope := copyScope(scope)
		lambdaScope[expr.Parameter] = true
		findings = walkForScopeViolations(findings, expr.Body, lambdaScope, path+".body", file)
		return findings
	}

	// For join_lookup, the fields map values are expressions, walk them
	if expr.Kind == "join_lookup" {
		for name, fieldExpr := range expr.Fields {
			fe := fieldExpr // avoid aliasing
			findings = walkForScopeViolations(findings, &fe, scope,
				fmt.Sprintf("%s.fields.%s", path, name), file)
		}
		return findings
	}

	// Recurse into sub-expressions
	findings = walkForScopeViolations(findings, expr.Object, scope, path+".object", file)
	findings = walkForScopeViolations(findings, expr.Left, scope, path+".left", file)
	findings = walkForScopeViolations(findings, expr.Right, scope, path+".right", file)
	findings = walkForScopeViolations(findings, expr.Target, scope, path+".target", file)
	findings = walkForScopeViolations(findings, expr.Operand, scope, path+".operand", file)
	findings = walkForScopeViolations(findings, expr.Collection, scope, path+".collection", file)
	findings = walkForScopeViolations(findings, expr.Lambda, scope, path+".lambda", file)
	findings = walkForScopeViolations(findings, expr.Condition, scope, path+".condition", file)
	findings = walkForScopeViolations(findings, expr.Body, scope, path+".body", file)
	findings = walkForScopeViolations(findings, expr.Element, scope, path+".element", file)

	for j := range expr.FuncArguments {
		findings = walkForScopeViolations(findings, &expr.FuncArguments[j], scope,
			fmt.Sprintf("%s.arguments[%d]", path, j), file)
	}
	for j := range expr.Elements {
		findings = walkForScopeViolations(findings, &expr.Elements[j], scope,
			fmt.Sprintf("%s.elements[%d]", path, j), file)
	}

	return findings
}

// walkEnsuresForScopeViolations walks an ensures clause tree for scope violations.
func walkEnsuresForScopeViolations(findings []report.Finding, ec ast.EnsuresClause, scope map[string]bool, path string, file string) []report.Finding {
	findings = walkForScopeViolations(findings, ec.Target, scope, path+".target", file)
	findings = walkForScopeViolations(findings, ec.Condition, scope, path+".condition", file)
	findings = walkForScopeViolations(findings, ec.Collection, scope, path+".collection", file)

	// Walk value if present (it's a json.RawMessage that could contain an Expression)
	if ec.Value != nil {
		var valExpr ast.Expression
		if err := json.Unmarshal(ec.Value, &valExpr); err == nil && valExpr.Kind != "" {
			findings = walkForScopeViolations(findings, &valExpr, scope, path+".value", file)
		}
	}

	// Walk fields map (entity_creation, trigger_emission)
	for name, fieldExpr := range ec.Fields {
		fe := fieldExpr
		findings = walkForScopeViolations(findings, &fe, scope,
			fmt.Sprintf("%s.fields.%s", path, name), file)
	}

	// Walk arguments map (trigger_emission)
	for name, argExpr := range ec.Arguments {
		ae := argExpr
		findings = walkForScopeViolations(findings, &ae, scope,
			fmt.Sprintf("%s.arguments.%s", path, name), file)
	}

	for j, then := range ec.Then {
		findings = walkEnsuresForScopeViolations(findings, then, scope,
			fmt.Sprintf("%s.then[%d]", path, j), file)
	}
	for j, el := range ec.Else {
		findings = walkEnsuresForScopeViolations(findings, el, scope,
			fmt.Sprintf("%s.else[%d]", path, j), file)
	}

	// iteration: add binding to scope for body
	if ec.Kind == "iteration" && ec.Binding != "" {
		iterScope := copyScope(scope)
		iterScope[ec.Binding] = true
		for j, body := range ec.Body {
			findings = walkEnsuresForScopeViolations(findings, body, iterScope,
				fmt.Sprintf("%s.body[%d]", path, j), file)
		}
	} else if ec.Kind == "let_binding" && ec.Binding != "" {
		// let_binding in ensures: add binding to scope for body
		// Walk the value with current scope
		if ec.Value != nil {
			var valExpr ast.Expression
			if err := json.Unmarshal(ec.Value, &valExpr); err == nil && valExpr.Kind != "" {
				// Already walked above, but let_binding value could be entity_creation
				// which is handled differently. The unmarshal above covers expression values.
			}
		}
		letScope := copyScope(scope)
		letScope[ec.Binding] = true
		for j, body := range ec.Body {
			findings = walkEnsuresForScopeViolations(findings, body, letScope,
				fmt.Sprintf("%s.body[%d]", path, j), file)
		}
	} else {
		for j, body := range ec.Body {
			findings = walkEnsuresForScopeViolations(findings, body, scope,
				fmt.Sprintf("%s.body[%d]", path, j), file)
		}
	}

	return findings
}

// --- RULE-12: Type mismatch checks ---

// resolveExprType returns a simple type descriptor string for an expression.
// Returns "" for unknown types — only known types are used in mismatch checks.
func resolveExprType(expr *ast.Expression, fieldTypes map[string]*ast.FieldType, st *SymbolTable) string {
	if expr == nil {
		return ""
	}

	switch expr.Kind {
	case "literal":
		return literalTypeToDescriptor(expr.Type)
	case "field_access":
		if expr.Object == nil {
			// Root field access: look up in entity fields
			if ft, ok := fieldTypes[expr.Field]; ok {
				return fieldTypeToDescriptor(ft)
			}
			return ""
		}
		// Chained field access: cannot resolve without full type tracking
		return ""
	case "arithmetic":
		// The result type of arithmetic is the common numeric/temporal type
		leftType := resolveExprType(expr.Left, fieldTypes, st)
		if leftType != "" {
			return leftType
		}
		return resolveExprType(expr.Right, fieldTypes, st)
	case "function_call":
		// Cannot determine return type of arbitrary functions
		return ""
	case "collection_op":
		if expr.Operation == "count" {
			return "Integer"
		}
		return ""
	default:
		return ""
	}
}

// literalTypeToDescriptor maps literal type strings to canonical type descriptors.
func literalTypeToDescriptor(litType string) string {
	switch litType {
	case "integer":
		return "Integer"
	case "string":
		return "String"
	case "boolean":
		return "Boolean"
	case "timestamp":
		return "Timestamp"
	case "duration":
		return "Duration"
	case "enum_value":
		return "EnumValue"
	case "null":
		return "Null"
	default:
		return ""
	}
}

// fieldTypeToDescriptor maps a FieldType to a canonical type descriptor.
func fieldTypeToDescriptor(ft *ast.FieldType) string {
	if ft == nil {
		return ""
	}
	switch ft.Kind {
	case "primitive":
		return ft.Value // "String", "Integer", "Boolean", "Timestamp", "Duration"
	case "inline_enum":
		return "InlineEnum"
	case "named_enum":
		return "NamedEnum:" + ft.Name
	case "optional":
		if ft.Inner != nil {
			return fieldTypeToDescriptor(ft.Inner)
		}
		return ""
	default:
		return ""
	}
}

// isNumericType returns true for types that can participate in arithmetic.
func isNumericType(t string) bool {
	return t == "Integer"
}

// isTemporalType returns true for Timestamp or Duration.
func isTemporalType(t string) bool {
	return t == "Timestamp" || t == "Duration"
}

// isComparable returns true if two known types can be compared.
func isComparable(left, right string) bool {
	if left == right {
		return true
	}
	// Null is comparable with anything (null checks)
	if left == "Null" || right == "Null" {
		return true
	}
	// EnumValue is comparable with enum types and strings
	if left == "EnumValue" || right == "EnumValue" {
		return true
	}
	// Temporal types are comparable with each other
	if isTemporalType(left) && isTemporalType(right) {
		return true
	}
	return false
}

// isValidArithmetic checks if an arithmetic expression has valid operand types.
// Returns (valid, leftType, rightType).
func isValidArithmetic(op string, leftType, rightType string) bool {
	// Integer arithmetic
	if isNumericType(leftType) && isNumericType(rightType) {
		return true
	}

	// Temporal arithmetic
	switch {
	case leftType == "Timestamp" && rightType == "Duration":
		return true // Timestamp +/- Duration
	case leftType == "Duration" && rightType == "Timestamp" && op == "+":
		return true // Duration + Timestamp
	case leftType == "Timestamp" && rightType == "Timestamp" && op == "-":
		return true // Timestamp - Timestamp = Duration
	case leftType == "Duration" && rightType == "Duration":
		return true // Duration +/- Duration
	}

	return false
}

// checkTypeMismatches validates type compatibility in comparisons and arithmetic.
func checkTypeMismatches(findings []report.Finding, spec *ast.Spec, st *SymbolTable) []report.Finding {
	for i, entity := range spec.Entities {
		fieldTypes := buildFieldTypeMap(entity.Fields)
		for j, dv := range entity.DerivedValues {
			findings = walkForTypeMismatches(findings, dv.Expression, fieldTypes, st,
				fmt.Sprintf("$.entities[%d].derived_values[%d].expression", i, j), spec.File)
		}
	}

	for i, rule := range spec.Rules {
		basePath := fmt.Sprintf("$.rules[%d]", i)
		fieldTypes := make(map[string]*ast.FieldType)
		if rule.Trigger.Entity != "" {
			if ent := st.LookupEntity(rule.Trigger.Entity); ent != nil {
				fieldTypes = buildFieldTypeMap(ent.Fields)
			}
		}

		for j, req := range rule.Requires {
			findings = walkForTypeMismatches(findings, &req, fieldTypes, st,
				fmt.Sprintf("%s.requires[%d]", basePath, j), spec.File)
		}

		for j, lb := range rule.LetBindings {
			findings = walkForTypeMismatches(findings, lb.Expression, fieldTypes, st,
				fmt.Sprintf("%s.let_bindings[%d].expression", basePath, j), spec.File)
		}

		for j, ec := range rule.Ensures {
			findings = walkEnsuresForTypeMismatches(findings, ec, fieldTypes, st,
				fmt.Sprintf("%s.ensures[%d]", basePath, j), spec.File)
		}
	}

	return findings
}

func walkForTypeMismatches(findings []report.Finding, expr *ast.Expression, fieldTypes map[string]*ast.FieldType, st *SymbolTable, path string, file string) []report.Finding {
	if expr == nil {
		return findings
	}

	if expr.Kind == "comparison" {
		leftType := resolveExprType(expr.Left, fieldTypes, st)
		rightType := resolveExprType(expr.Right, fieldTypes, st)

		if leftType != "" && rightType != "" && !isComparable(leftType, rightType) {
			findings = append(findings, report.NewError(
				"RULE-12",
				fmt.Sprintf("Type mismatch in comparison: %s vs %s", leftType, rightType),
				report.Location{File: file, Path: path},
			))
		}
	}

	if expr.Kind == "arithmetic" {
		leftType := resolveExprType(expr.Left, fieldTypes, st)
		rightType := resolveExprType(expr.Right, fieldTypes, st)

		if leftType != "" && rightType != "" && !isValidArithmetic(expr.Operator, leftType, rightType) {
			// Determine which side is the non-numeric/non-temporal one
			if !isNumericType(leftType) && !isTemporalType(leftType) {
				findings = append(findings, report.NewError(
					"RULE-12",
					fmt.Sprintf("Non-numeric type %s in arithmetic", leftType),
					report.Location{File: file, Path: path},
				))
			} else if !isNumericType(rightType) && !isTemporalType(rightType) {
				findings = append(findings, report.NewError(
					"RULE-12",
					fmt.Sprintf("Non-numeric type %s in arithmetic", rightType),
					report.Location{File: file, Path: path},
				))
			} else {
				// Both are numeric/temporal but the combination is invalid
				findings = append(findings, report.NewError(
					"RULE-12",
					fmt.Sprintf("Type mismatch in arithmetic: %s %s %s", leftType, expr.Operator, rightType),
					report.Location{File: file, Path: path},
				))
			}
		}
	}

	// Recurse
	findings = walkForTypeMismatches(findings, expr.Object, fieldTypes, st, path+".object", file)
	findings = walkForTypeMismatches(findings, expr.Left, fieldTypes, st, path+".left", file)
	findings = walkForTypeMismatches(findings, expr.Right, fieldTypes, st, path+".right", file)
	findings = walkForTypeMismatches(findings, expr.Target, fieldTypes, st, path+".target", file)
	findings = walkForTypeMismatches(findings, expr.Operand, fieldTypes, st, path+".operand", file)
	findings = walkForTypeMismatches(findings, expr.Collection, fieldTypes, st, path+".collection", file)
	findings = walkForTypeMismatches(findings, expr.Lambda, fieldTypes, st, path+".lambda", file)
	findings = walkForTypeMismatches(findings, expr.Condition, fieldTypes, st, path+".condition", file)
	findings = walkForTypeMismatches(findings, expr.Body, fieldTypes, st, path+".body", file)
	findings = walkForTypeMismatches(findings, expr.Element, fieldTypes, st, path+".element", file)

	for j := range expr.FuncArguments {
		findings = walkForTypeMismatches(findings, &expr.FuncArguments[j], fieldTypes, st,
			fmt.Sprintf("%s.arguments[%d]", path, j), file)
	}
	for j := range expr.Elements {
		findings = walkForTypeMismatches(findings, &expr.Elements[j], fieldTypes, st,
			fmt.Sprintf("%s.elements[%d]", path, j), file)
	}

	return findings
}

func walkEnsuresForTypeMismatches(findings []report.Finding, ec ast.EnsuresClause, fieldTypes map[string]*ast.FieldType, st *SymbolTable, path string, file string) []report.Finding {
	findings = walkForTypeMismatches(findings, ec.Target, fieldTypes, st, path+".target", file)
	findings = walkForTypeMismatches(findings, ec.Condition, fieldTypes, st, path+".condition", file)
	findings = walkForTypeMismatches(findings, ec.Collection, fieldTypes, st, path+".collection", file)

	if ec.Value != nil {
		var valExpr ast.Expression
		if err := json.Unmarshal(ec.Value, &valExpr); err == nil && valExpr.Kind != "" {
			findings = walkForTypeMismatches(findings, &valExpr, fieldTypes, st, path+".value", file)
		}
	}

	for name, fieldExpr := range ec.Fields {
		fe := fieldExpr
		findings = walkForTypeMismatches(findings, &fe, fieldTypes, st,
			fmt.Sprintf("%s.fields.%s", path, name), file)
	}

	for j, then := range ec.Then {
		findings = walkEnsuresForTypeMismatches(findings, then, fieldTypes, st,
			fmt.Sprintf("%s.then[%d]", path, j), file)
	}
	for j, el := range ec.Else {
		findings = walkEnsuresForTypeMismatches(findings, el, fieldTypes, st,
			fmt.Sprintf("%s.else[%d]", path, j), file)
	}
	for j, body := range ec.Body {
		findings = walkEnsuresForTypeMismatches(findings, body, fieldTypes, st,
			fmt.Sprintf("%s.body[%d]", path, j), file)
	}

	return findings
}

// --- RULE-13: any/all lambda check ---

func checkCollectionOps(findings []report.Finding, spec *ast.Spec) []report.Finding {
	for i, entity := range spec.Entities {
		for j, dv := range entity.DerivedValues {
			findings = walkForCollectionOps(findings, dv.Expression,
				fmt.Sprintf("$.entities[%d].derived_values[%d].expression", i, j), spec.File)
		}
	}
	for i, rule := range spec.Rules {
		basePath := fmt.Sprintf("$.rules[%d]", i)
		for j, req := range rule.Requires {
			findings = walkForCollectionOps(findings, &req,
				fmt.Sprintf("%s.requires[%d]", basePath, j), spec.File)
		}
		for j, ec := range rule.Ensures {
			findings = walkEnsuresForCollectionOps(findings, ec,
				fmt.Sprintf("%s.ensures[%d]", basePath, j), spec.File)
		}
	}
	return findings
}

func walkForCollectionOps(findings []report.Finding, expr *ast.Expression, path string, file string) []report.Finding {
	if expr == nil {
		return findings
	}

	if expr.Kind == "collection_op" {
		op := expr.Operation
		if op == "any" || op == "all" {
			if expr.Lambda == nil || expr.Lambda.Kind != "lambda" || expr.Lambda.Parameter == "" {
				findings = append(findings, report.NewError(
					"RULE-13",
					fmt.Sprintf("Collection operation '%s' requires explicit lambda parameter", op),
					report.Location{File: file, Path: path},
				))
			}
		}
	}

	// Recurse
	findings = walkForCollectionOps(findings, expr.Object, path+".object", file)
	findings = walkForCollectionOps(findings, expr.Left, path+".left", file)
	findings = walkForCollectionOps(findings, expr.Right, path+".right", file)
	findings = walkForCollectionOps(findings, expr.Target, path+".target", file)
	findings = walkForCollectionOps(findings, expr.Operand, path+".operand", file)
	findings = walkForCollectionOps(findings, expr.Collection, path+".collection", file)
	findings = walkForCollectionOps(findings, expr.Lambda, path+".lambda", file)
	findings = walkForCollectionOps(findings, expr.Condition, path+".condition", file)
	findings = walkForCollectionOps(findings, expr.Body, path+".body", file)
	findings = walkForCollectionOps(findings, expr.Element, path+".element", file)
	for j := range expr.FuncArguments {
		findings = walkForCollectionOps(findings, &expr.FuncArguments[j],
			fmt.Sprintf("%s.arguments[%d]", path, j), file)
	}
	for j := range expr.Elements {
		findings = walkForCollectionOps(findings, &expr.Elements[j],
			fmt.Sprintf("%s.elements[%d]", path, j), file)
	}

	return findings
}

func walkEnsuresForCollectionOps(findings []report.Finding, ec ast.EnsuresClause, path string, file string) []report.Finding {
	findings = walkForCollectionOps(findings, ec.Target, path+".target", file)
	findings = walkForCollectionOps(findings, ec.Condition, path+".condition", file)
	findings = walkForCollectionOps(findings, ec.Collection, path+".collection", file)

	for j, then := range ec.Then {
		findings = walkEnsuresForCollectionOps(findings, then,
			fmt.Sprintf("%s.then[%d]", path, j), file)
	}
	for j, el := range ec.Else {
		findings = walkEnsuresForCollectionOps(findings, el,
			fmt.Sprintf("%s.else[%d]", path, j), file)
	}
	for j, body := range ec.Body {
		findings = walkEnsuresForCollectionOps(findings, body,
			fmt.Sprintf("%s.body[%d]", path, j), file)
	}

	return findings
}

// --- RULE-14: Enum comparison checks ---

func checkEnumComparisons(findings []report.Finding, spec *ast.Spec, st *SymbolTable) []report.Finding {
	for i, entity := range spec.Entities {
		// Build a map of field name -> type for this entity
		fieldTypes := buildFieldTypeMap(entity.Fields)

		for j, dv := range entity.DerivedValues {
			findings = walkForEnumComparisons(findings, dv.Expression, fieldTypes, st,
				fmt.Sprintf("$.entities[%d].derived_values[%d].expression", i, j), spec.File)
		}
	}

	for i, rule := range spec.Rules {
		basePath := fmt.Sprintf("$.rules[%d]", i)

		// Build scope field types from trigger entity
		fieldTypes := make(map[string]*ast.FieldType)
		if rule.Trigger.Entity != "" {
			if ent := st.LookupEntity(rule.Trigger.Entity); ent != nil {
				fieldTypes = buildFieldTypeMap(ent.Fields)
			}
		}

		for j, req := range rule.Requires {
			findings = walkForEnumComparisons(findings, &req, fieldTypes, st,
				fmt.Sprintf("%s.requires[%d]", basePath, j), spec.File)
		}
	}

	return findings
}

func buildFieldTypeMap(fields []ast.Field) map[string]*ast.FieldType {
	m := make(map[string]*ast.FieldType, len(fields))
	for i := range fields {
		m[fields[i].Name] = &fields[i].Type
	}
	return m
}

func walkForEnumComparisons(findings []report.Finding, expr *ast.Expression, fieldTypes map[string]*ast.FieldType, st *SymbolTable, path string, file string) []report.Finding {
	if expr == nil {
		return findings
	}

	if expr.Kind == "comparison" {
		leftType := resolveExprEnumType(expr.Left, fieldTypes, st)
		rightType := resolveExprEnumType(expr.Right, fieldTypes, st)

		if leftType != nil && rightType != nil {
			// Both sides are enum-typed
			if leftType.Kind == "inline_enum" || rightType.Kind == "inline_enum" {
				// Any inline enum comparison across different fields is invalid
				findings = append(findings, report.NewError(
					"RULE-14",
					"Cannot compare inline enums from different fields",
					report.Location{File: file, Path: path},
				))
			} else if leftType.Kind == "named_enum" && rightType.Kind == "named_enum" {
				if leftType.Name != rightType.Name {
					findings = append(findings, report.NewError(
						"RULE-14",
						fmt.Sprintf("Cannot compare named enums of different types: '%s' vs '%s'", leftType.Name, rightType.Name),
						report.Location{File: file, Path: path},
					))
				}
			}
		}
	}

	// Recurse
	findings = walkForEnumComparisons(findings, expr.Object, fieldTypes, st, path+".object", file)
	findings = walkForEnumComparisons(findings, expr.Left, fieldTypes, st, path+".left", file)
	findings = walkForEnumComparisons(findings, expr.Right, fieldTypes, st, path+".right", file)
	findings = walkForEnumComparisons(findings, expr.Target, fieldTypes, st, path+".target", file)
	findings = walkForEnumComparisons(findings, expr.Operand, fieldTypes, st, path+".operand", file)
	findings = walkForEnumComparisons(findings, expr.Collection, fieldTypes, st, path+".collection", file)
	findings = walkForEnumComparisons(findings, expr.Lambda, fieldTypes, st, path+".lambda", file)
	findings = walkForEnumComparisons(findings, expr.Condition, fieldTypes, st, path+".condition", file)
	findings = walkForEnumComparisons(findings, expr.Body, fieldTypes, st, path+".body", file)
	findings = walkForEnumComparisons(findings, expr.Element, fieldTypes, st, path+".element", file)

	return findings
}

// resolveExprEnumType tries to determine if an expression resolves to an enum type.
func resolveExprEnumType(expr *ast.Expression, fieldTypes map[string]*ast.FieldType, _ *SymbolTable) *ast.FieldType {
	if expr == nil {
		return nil
	}
	if expr.Kind == "field_access" && expr.Object == nil {
		// Root field access -- look up in entity fields
		if ft, ok := fieldTypes[expr.Field]; ok {
			if ft.Kind == "inline_enum" || ft.Kind == "named_enum" {
				return ft
			}
		}
	}
	return nil
}
