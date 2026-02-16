// Package checker orchestrates schema and semantic validation passes for
// Allium specification files, producing a consolidated report.
package checker

import (
	"fmt"
	"os"
	"slices"

	"github.com/foundry-zero/allium/internal/ast"
	"github.com/foundry-zero/allium/internal/report"
	"github.com/foundry-zero/allium/internal/schema"
	"github.com/foundry-zero/allium/internal/semantic"
)

// PassFunc is a semantic validation pass that inspects a parsed spec
// and returns any findings (errors or warnings).
type PassFunc func(*ast.Spec, *semantic.SymbolTable) []report.Finding

// CheckOptions controls which validation passes to run.
type CheckOptions struct {
	SchemaOnly bool  // Only run JSON Schema validation, skip semantic passes.
	RuleFilter []int // If non-empty, only run passes covering these rule numbers.
	Strict     bool  // Treat warnings as errors for exit-code purposes.
}

// passEntry binds a named semantic pass to the rule numbers it covers.
type passEntry struct {
	Name  string
	Rules []int
	Fn    PassFunc
}

// Checker orchestrates validation of .allium.json files.
type Checker struct {
	sv     *schema.SchemaValidator
	passes []passEntry
}

// NewChecker creates a Checker with the embedded JSON Schema validator
// and all available semantic passes registered.
func NewChecker() (*Checker, error) {
	sv, err := schema.NewSchemaValidator()
	if err != nil {
		return nil, fmt.Errorf("initialize schema validator: %w", err)
	}
	c := &Checker{sv: sv}
	registerPasses(c)
	return c, nil
}

// RegisterPass adds a semantic validation pass to the checker.
// It is typically called from registerPasses during initialization.
func (c *Checker) RegisterPass(name string, rules []int, fn PassFunc) {
	c.passes = append(c.passes, passEntry{Name: name, Rules: rules, Fn: fn})
}

// Check validates the Allium spec file at path and returns a report.
// It runs schema validation first, then semantic passes (if the schema is valid
// and SchemaOnly is not set).
func (c *Checker) Check(path string, opts CheckOptions) *report.Report {
	r := report.NewReport(path)

	// Verify the file is accessible before attempting validation.
	if _, err := os.Stat(path); err != nil {
		r.AddFinding(report.NewError("INPUT", fmt.Sprintf("cannot access file: %v", err),
			report.Location{File: path}))
		return r
	}

	// --- Phase 1: JSON Schema validation ---
	schemaErrors := c.sv.Validate(path)
	r.SchemaValid = len(schemaErrors) == 0

	for _, se := range schemaErrors {
		r.AddFinding(report.NewError("SCHEMA", se.Message,
			report.Location{File: path, Path: se.Path}))
	}

	if !r.SchemaValid || opts.SchemaOnly {
		return r
	}

	// --- Phase 2: Load AST ---
	spec, err := ast.LoadSpec(path)
	if err != nil {
		r.AddFinding(report.NewError("INPUT", fmt.Sprintf("failed to load spec: %v", err),
			report.Location{File: path}))
		return r
	}

	// --- Phase 3: Build symbol table ---
	st := semantic.BuildSymbolTable(spec)

	// --- Phase 4: Run semantic passes ---
	for _, p := range c.passes {
		if !passMatchesFilter(p.Rules, opts.RuleFilter) {
			continue
		}
		findings := p.Fn(spec, st)
		for _, f := range findings {
			r.AddFinding(f)
		}
	}

	return r
}

// passMatchesFilter returns true if any of the pass's rules are in the filter,
// or if the filter is empty (meaning run all passes).
func passMatchesFilter(passRules []int, filter []int) bool {
	if len(filter) == 0 {
		return true
	}
	for _, pr := range passRules {
		if slices.Contains(filter, pr) {
			return true
		}
	}
	return false
}

// registerPasses wires up all available semantic passes.
func registerPasses(c *Checker) {
	c.RegisterPass("references", []int{1, 3, 22, 27, 28, 30, 31, 35}, semantic.CheckReferences)
	c.RegisterPass("uniqueness", []int{6, 23, 26}, semantic.CheckUniqueness)
	c.RegisterPass("statemachines", []int{7, 8, 9}, semantic.CheckStateMachines)
	c.RegisterPass("expressions", []int{10, 11, 12, 13, 14}, semantic.CheckExpressions)
	c.RegisterPass("sumtypes", []int{16, 17, 18, 19}, semantic.CheckSumTypes)
	c.RegisterPass("surfaces", []int{29, 32, 33, 34}, semantic.CheckSurfaces)
	c.RegisterPass("warnings", nil, semantic.CheckWarnings)
}
