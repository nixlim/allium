package semantic

import (
	"fmt"

	"github.com/foundry-zero/allium/internal/ast"
	"github.com/foundry-zero/allium/internal/report"
)

// CheckUniqueness verifies that declarations which must be unique are not duplicated.
//
//   - RULE-06: Rules sharing a trigger name must have compatible parameters
//   - RULE-23: Given binding names must be unique
//   - RULE-26: Config parameter names must be unique
func CheckUniqueness(spec *ast.Spec, st *SymbolTable) []report.Finding {
	var findings []report.Finding

	findings = checkTriggerCompatibility(findings, spec, st)
	findings = checkGivenUniqueness(findings, spec)
	findings = checkConfigUniqueness(findings, spec)

	return findings
}

// checkTriggerCompatibility checks RULE-06: rules sharing an external_stimulus or
// chained trigger name must have the same parameter count and names.
func checkTriggerCompatibility(findings []report.Finding, spec *ast.Spec, st *SymbolTable) []report.Finding {
	for triggerName, rules := range st.Triggers {
		if len(rules) < 2 {
			continue
		}

		// Use first rule as reference
		ref := rules[0]
		refParams := ref.Trigger.Parameters

		for k := 1; k < len(rules); k++ {
			other := rules[k]
			otherParams := other.Trigger.Parameters

			if len(refParams) != len(otherParams) {
				findings = append(findings, report.NewError(
					"RULE-06",
					fmt.Sprintf(
						"Rules sharing trigger '%s' have incompatible parameters: '%s' has %d but '%s' has %d",
						triggerName, ref.Name, len(refParams), other.Name, len(otherParams),
					),
					report.Location{
						File: spec.File,
						Path: fmt.Sprintf("$.rules[?(@.name=='%s')].trigger", other.Name),
					},
				))
				continue
			}

			// Check parameter name compatibility
			for p := range len(refParams) {
				if refParams[p].Name != otherParams[p].Name {
					findings = append(findings, report.NewError(
						"RULE-06",
						fmt.Sprintf(
							"Rules sharing trigger '%s' have incompatible parameter at position %d: '%s' uses '%s' but '%s' uses '%s'",
							triggerName, p, ref.Name, refParams[p].Name, other.Name, otherParams[p].Name,
						),
						report.Location{
							File: spec.File,
							Path: fmt.Sprintf("$.rules[?(@.name=='%s')].trigger.parameters[%d]", other.Name, p),
						},
					))
				}
			}
		}
	}
	return findings
}

// checkGivenUniqueness checks RULE-23: no duplicate given binding names.
func checkGivenUniqueness(findings []report.Finding, spec *ast.Spec) []report.Finding {
	seen := make(map[string]int, len(spec.Given))
	for i, g := range spec.Given {
		if prev, ok := seen[g.Name]; ok {
			findings = append(findings, report.NewError(
				"RULE-23",
				fmt.Sprintf("Duplicate given binding name '%s' (first at index %d)", g.Name, prev),
				report.Location{File: spec.File, Path: fmt.Sprintf("$.given[%d]", i)},
			))
		} else {
			seen[g.Name] = i
		}
	}
	return findings
}

// checkConfigUniqueness checks RULE-26: no duplicate config parameter names.
func checkConfigUniqueness(findings []report.Finding, spec *ast.Spec) []report.Finding {
	seen := make(map[string]int, len(spec.Config))
	for i, c := range spec.Config {
		if prev, ok := seen[c.Name]; ok {
			findings = append(findings, report.NewError(
				"RULE-26",
				fmt.Sprintf("Duplicate config parameter name '%s' (first at index %d)", c.Name, prev),
				report.Location{File: spec.File, Path: fmt.Sprintf("$.config[%d]", i)},
			))
		} else {
			seen[c.Name] = i
		}
	}
	return findings
}
