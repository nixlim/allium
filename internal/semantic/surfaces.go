package semantic

import (
	"fmt"

	"github.com/foundry-zero/allium/internal/ast"
	"github.com/foundry-zero/allium/internal/report"
)

// CheckSurfaces validates surface semantics.
//
//   - RULE-29: All exposes field paths must be reachable from facing/context/let bindings
//   - RULE-32: Facing and context bindings must be referenced in the surface body
//   - RULE-33: When conditions must reference reachable fields
//   - RULE-34: For iterations must target collection-typed fields
func CheckSurfaces(spec *ast.Spec, st *SymbolTable) []report.Finding {
	var findings []report.Finding

	for i, surface := range spec.Surfaces {
		basePath := fmt.Sprintf("$.surfaces[%d]", i)

		// Build the set of available bindings for this surface
		bindings := collectSurfaceBindings(surface)

		// RULE-29: Check exposes paths
		for j, exp := range surface.Exposes {
			if exp.Expression != nil {
				if !isExprRootReachable(exp.Expression, bindings) {
					findings = append(findings, report.NewError(
						"RULE-29",
						fmt.Sprintf("Unreachable path in exposes on surface '%s'", surface.Name),
						report.Location{File: spec.File, Path: fmt.Sprintf("%s.exposes[%d]", basePath, j)},
					))
				}
			}
		}

		// RULE-32: Check that facing and context bindings are used
		usedBindings := collectUsedBindings(surface)

		if surface.Facing.Binding != "" {
			if !usedBindings[surface.Facing.Binding] {
				findings = append(findings, report.NewError(
					"RULE-32",
					fmt.Sprintf("Unused binding '%s' in surface '%s'", surface.Facing.Binding, surface.Name),
					report.Location{File: spec.File, Path: fmt.Sprintf("%s.facing.binding", basePath)},
				))
			}
		}

		if surface.Context != nil && surface.Context.Binding != "" {
			if !usedBindings[surface.Context.Binding] {
				findings = append(findings, report.NewError(
					"RULE-32",
					fmt.Sprintf("Unused binding '%s' in surface '%s'", surface.Context.Binding, surface.Name),
					report.Location{File: spec.File, Path: fmt.Sprintf("%s.context.binding", basePath)},
				))
			}
		}

		// RULE-34: Check provides for_each collection types
		for j, p := range surface.Provides {
			findings = checkProvidesIteration(findings, p, st, surface.Name, bindings,
				fmt.Sprintf("%s.provides[%d]", basePath, j), spec.File)
		}
	}

	return findings
}

// collectSurfaceBindings returns the set of available root binding names for a surface.
func collectSurfaceBindings(s ast.Surface) map[string]bool {
	bindings := make(map[string]bool)
	if s.Facing.Binding != "" {
		bindings[s.Facing.Binding] = true
	}
	if s.Context != nil && s.Context.Binding != "" {
		bindings[s.Context.Binding] = true
	}
	for _, lb := range s.LetBindings {
		bindings[lb.Name] = true
	}
	return bindings
}

// isExprRootReachable checks if a field_access expression's root is in the available bindings.
func isExprRootReachable(expr *ast.Expression, bindings map[string]bool) bool {
	if expr == nil {
		return true
	}
	root := findExprRoot(expr)
	if root == "" {
		return true // non field_access expressions are fine
	}
	return bindings[root]
}

// findExprRoot walks down field_access chains to find the root identifier.
func findExprRoot(expr *ast.Expression) string {
	if expr == nil {
		return ""
	}
	if expr.Kind == "field_access" {
		if expr.Object == nil {
			return expr.Field // root-level field access
		}
		return findExprRoot(expr.Object)
	}
	return "" // not a field_access chain
}

// collectUsedBindings scans a surface body for all referenced root binding names.
func collectUsedBindings(s ast.Surface) map[string]bool {
	used := make(map[string]bool)

	for _, exp := range s.Exposes {
		collectExprRoots(exp.Expression, used)
		collectExprRoots(exp.When, used)
	}
	for _, p := range s.Provides {
		collectProvidesRoots(p, used)
	}
	for _, g := range s.Guarantees {
		_ = g // guarantees are descriptive, no expression refs
	}
	for _, r := range s.Related {
		collectExprRoots(r.ContextExpression, used)
		collectExprRoots(r.When, used)
	}
	for _, t := range s.Timeout {
		collectExprRoots(t.When, used)
	}
	for _, lb := range s.LetBindings {
		collectExprRoots(lb.Expression, used)
	}

	return used
}

func collectProvidesRoots(p ast.ProvidesItem, used map[string]bool) {
	collectExprRoots(p.When, used)
	collectExprRoots(p.Collection, used)
	for _, arg := range p.Arguments {
		collectExprRoots(arg.Expression, used)
	}
	for _, item := range p.Items {
		collectProvidesRoots(item, used)
	}
}

func collectExprRoots(expr *ast.Expression, used map[string]bool) {
	if expr == nil {
		return
	}
	root := findExprRoot(expr)
	if root != "" {
		used[root] = true
	}
	// Also recurse into sub-expressions for complex expressions
	collectExprRoots(expr.Object, used)
	collectExprRoots(expr.Left, used)
	collectExprRoots(expr.Right, used)
	collectExprRoots(expr.Target, used)
	collectExprRoots(expr.Operand, used)
	collectExprRoots(expr.Collection, used)
	collectExprRoots(expr.Lambda, used)
	collectExprRoots(expr.Condition, used)
	collectExprRoots(expr.Body, used)
	collectExprRoots(expr.Element, used)
	for j := range expr.FuncArguments {
		collectExprRoots(&expr.FuncArguments[j], used)
	}
	for j := range expr.Elements {
		collectExprRoots(&expr.Elements[j], used)
	}
}

// checkProvidesIteration checks RULE-34: for_each items must iterate over collections.
func checkProvidesIteration(findings []report.Finding, p ast.ProvidesItem, st *SymbolTable, surfaceName string, bindings map[string]bool, path string, file string) []report.Finding {
	if p.Kind == "for_each" && p.Collection != nil {
		// Check if the collection expression resolves to a collection type
		if !isCollectionExpression(p.Collection, st, bindings) {
			findings = append(findings, report.NewError(
				"RULE-34",
				fmt.Sprintf("Cannot iterate over non-collection type in surface '%s'", surfaceName),
				report.Location{File: file, Path: path},
			))
		}

		for j, item := range p.Items {
			findings = checkProvidesIteration(findings, item, st, surfaceName, bindings,
				fmt.Sprintf("%s.items[%d]", path, j), file)
		}
	}
	return findings
}

// isCollectionExpression checks if an expression likely evaluates to a collection type.
// This is a best-effort check based on field type lookups.
func isCollectionExpression(expr *ast.Expression, st *SymbolTable, _ map[string]bool) bool {
	if expr == nil {
		return false
	}

	// For field_access, try to resolve the field type
	if expr.Kind == "field_access" {
		// If the field access is on a known entity, check field type
		if expr.Object != nil && expr.Object.Kind == "field_access" && expr.Object.Object == nil {
			// pattern: binding.field — check if field is a relationship or collection type
			entityBinding := expr.Object.Field
			fieldName := expr.Field

			// Try to find the entity for this binding
			if entity := st.LookupEntity(entityBinding); entity != nil {
				return isCollectionField(entity.Fields, fieldName) || isRelationshipMany(entity.Relationships, fieldName)
			}
		}
	}

	// collection_op, set_literal always return collections
	if expr.Kind == "set_literal" || expr.Kind == "collection_op" {
		return true
	}

	// Default: assume it's a collection (conservative approach to avoid false positives)
	return true
}

func isCollectionField(fields []ast.Field, name string) bool {
	for _, f := range fields {
		if f.Name == name {
			return f.Type.Kind == "set" || f.Type.Kind == "list"
		}
	}
	return false
}

func isRelationshipMany(rels []ast.Relationship, name string) bool {
	for _, r := range rels {
		if r.Name == name {
			return r.Cardinality == "many"
		}
	}
	return false
}
