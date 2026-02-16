package semantic

import (
	"fmt"
	"strings"

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

	// Build reverse index: base entity name -> variant names from declarations
	variantsByBase := make(map[string][]string)
	for _, v := range spec.Variants {
		variantsByBase[v.BaseEntity] = append(variantsByBase[v.BaseEntity], v.Name)
	}

	// Build map: base entity name -> discriminator info.
	// An entity has a discriminator if variants point to it AND it has an
	// inline_enum field whose values correspond to variant names.
	discriminators := make(map[string]*discInfo)

	for i, entity := range spec.Entities {
		variantNames, hasVariants := variantsByBase[entity.Name]
		if !hasVariants {
			continue
		}
		for _, f := range entity.Fields {
			if f.Type.Kind == "inline_enum" && isDiscriminatorField(f.Type.Values, variantNames) {
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
			v := lookupVariantByEnumValue(st, variantName)
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

		if !containsVariantName(disc.variants, v.Name) {
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

// isDiscriminatorField checks if an inline_enum field serves as a discriminator
// by testing whether any of its values correspond to known variant names.
// Handles both exact match (PascalCase values) and snake_case → PascalCase conversion.
func isDiscriminatorField(enumValues []string, variantNames []string) bool {
	if len(variantNames) == 0 || len(enumValues) == 0 {
		return false
	}
	for _, vn := range variantNames {
		for _, ev := range enumValues {
			if ev == vn || snakeToPascal(ev) == vn {
				return true
			}
		}
	}
	return false
}

// lookupVariantByEnumValue looks up a variant declaration by an enum value.
// Tries exact match first, then snake_case → PascalCase conversion.
func lookupVariantByEnumValue(st *SymbolTable, enumValue string) *ast.Variant {
	if v := st.LookupVariant(enumValue); v != nil {
		return v
	}
	return st.LookupVariant(snakeToPascal(enumValue))
}

// containsVariantName checks if a variant name appears in a list of enum values,
// handling both exact match and snake_case → PascalCase comparison.
func containsVariantName(enumValues []string, variantName string) bool {
	for _, ev := range enumValues {
		if ev == variantName || snakeToPascal(ev) == variantName {
			return true
		}
	}
	return false
}

// snakeToPascal converts a snake_case string to PascalCase.
func snakeToPascal(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
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
