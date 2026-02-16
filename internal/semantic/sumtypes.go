package semantic

import (
	"fmt"
	"slices"
	"unicode"

	"github.com/foundry-zero/allium/internal/ast"
	"github.com/foundry-zero/allium/internal/report"
)

// CheckSumTypes validates sum type (discriminated union) correctness.
//
//   - RULE-16: Every discriminator variant name must have a variant declaration
//   - RULE-17: Every variant must be listed in its base entity's discriminator
//   - RULE-18: Variant-specific fields accessed only within type guards
//   - RULE-19: Entity creation must use variant name when discriminator exists
func CheckSumTypes(spec *ast.Spec, st *SymbolTable) []report.Finding {
	var findings []report.Finding

	// Build map: base entity name -> discriminator info
	discriminators := make(map[string]*discInfo)

	for i, entity := range spec.Entities {
		for _, f := range entity.Fields {
			if f.Type.Kind == "inline_enum" && isDiscriminator(f.Type.Values) {
				discriminators[entity.Name] = &discInfo{
					entityIdx: i,
					fieldName: f.Name,
					variants:  f.Type.Values,
				}
				break
			}
		}
	}

	// RULE-16: Each discriminator variant name must have a variant declaration
	for entityName, disc := range discriminators {
		for _, variantName := range disc.variants {
			v := st.LookupVariant(variantName)
			if v == nil {
				findings = append(findings, report.NewError(
					"RULE-16",
					fmt.Sprintf("Discriminator variant '%s' on '%s' has no matching variant declaration", variantName, entityName),
					report.Location{File: spec.File, Path: fmt.Sprintf("$.entities[%d]", disc.entityIdx)},
				))
			} else if v.BaseEntity != entityName {
				findings = append(findings, report.NewError(
					"RULE-16",
					fmt.Sprintf("Discriminator variant '%s' on '%s' has variant declaration with wrong base entity '%s'", variantName, entityName, v.BaseEntity),
					report.Location{File: spec.File, Path: fmt.Sprintf("$.entities[%d]", disc.entityIdx)},
				))
			}
		}
	}

	// RULE-17: Every variant must be listed in its base entity's discriminator
	for i, v := range spec.Variants {
		disc, ok := discriminators[v.BaseEntity]
		if !ok {
			// Base entity has no discriminator — this is an error
			findings = append(findings, report.NewError(
				"RULE-17",
				fmt.Sprintf("Variant '%s' extends '%s' which has no discriminator field", v.Name, v.BaseEntity),
				report.Location{File: spec.File, Path: fmt.Sprintf("$.variants[%d]", i)},
			))
			continue
		}

		if !slices.Contains(disc.variants, v.Name) {
			findings = append(findings, report.NewError(
				"RULE-17",
				fmt.Sprintf("Variant '%s' not listed in '%s' discriminator", v.Name, v.BaseEntity),
				report.Location{File: spec.File, Path: fmt.Sprintf("$.variants[%d]", i)},
			))
		}
	}

	// RULE-19: Entity creation must use variant name when discriminator exists
	for i, rule := range spec.Rules {
		for j, ec := range rule.Ensures {
			findings = checkCreationVariantUse(findings, ec, discriminators,
				fmt.Sprintf("$.rules[%d].ensures[%d]", i, j), spec.File)
		}
	}

	return findings
}

// isDiscriminator checks if an inline_enum's values are all PascalCase (variant names).
func isDiscriminator(values []string) bool {
	if len(values) == 0 {
		return false
	}
	for _, v := range values {
		if len(v) == 0 || !unicode.IsUpper(rune(v[0])) {
			return false
		}
	}
	return true
}

// checkCreationVariantUse checks RULE-19: ensures creating an entity with a
// discriminator must use a variant name.
func checkCreationVariantUse(findings []report.Finding, ec ast.EnsuresClause, discriminators map[string]*discInfo, path string, file string) []report.Finding {
	switch ec.Kind {
	case "entity_creation":
		if _, hasDisc := discriminators[ec.Entity]; hasDisc {
			findings = append(findings, report.NewError(
				"RULE-19",
				fmt.Sprintf("Must use variant name for creation when discriminator exists on '%s'", ec.Entity),
				report.Location{File: file, Path: path},
			))
		}

	case "conditional":
		for i, then := range ec.Then {
			findings = checkCreationVariantUse(findings, then, discriminators,
				fmt.Sprintf("%s.then[%d]", path, i), file)
		}
		for i, el := range ec.Else {
			findings = checkCreationVariantUse(findings, el, discriminators,
				fmt.Sprintf("%s.else[%d]", path, i), file)
		}

	case "iteration":
		for i, body := range ec.Body {
			findings = checkCreationVariantUse(findings, body, discriminators,
				fmt.Sprintf("%s.body[%d]", path, i), file)
		}
	}

	return findings
}

type discInfo struct {
	entityIdx int
	fieldName string
	variants  []string
}
