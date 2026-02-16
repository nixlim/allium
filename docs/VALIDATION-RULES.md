# Allium Validation Rules Reference

This document is the master index for all validation rules and warnings enforced by `allium-check`.

Rules are errors (exit code 1). Warnings are advisory (exit code 0 unless `--strict`).

## Rules by Group

| Group | Rules | Documentation |
|-------|-------|---------------|
| Structural (schema-enforced) | RULE-02, 04, 05, 15, 20, 21, 24, 25 | [structural.md](rules/structural.md) |
| Reference Resolution | RULE-01, 03, 22, 27, 28, 30, 31, 35 | [reference.md](rules/reference.md) |
| Uniqueness | RULE-06, 23, 26 | [uniqueness.md](rules/uniqueness.md) |
| State Machine | RULE-07, 08, 09 | [state-machine.md](rules/state-machine.md) |
| Expression | RULE-10, 11, 12, 13, 14 | [expression.md](rules/expression.md) |
| Sum Type | RULE-16, 17, 18, 19 | [sum-type.md](rules/sum-type.md) |
| Surface | RULE-29, 32, 33, 34 | [surface.md](rules/surface.md) |

## All Rules

| ID | Severity | Description | Group |
|----|----------|-------------|-------|
| RULE-01 | error | Entity referenced but not declared | Reference |
| RULE-02 | error | Every field must declare a type | Structural |
| RULE-03 | error | Relationship target entity not declared | Reference |
| RULE-04 | error | Every rule must have a trigger and non-empty ensures | Structural |
| RULE-05 | error | Trigger kind must be one of 7 valid kinds | Structural |
| RULE-06 | error | Rules sharing a trigger must have compatible parameters | Uniqueness |
| RULE-07 | error | Unreachable status enum value | State Machine |
| RULE-08 | error | Dead-end state with no outgoing transition | State Machine |
| RULE-09 | error | Undeclared status value in assignment | State Machine |
| RULE-10 | error | Cycle detected in derived value dependencies | Expression |
| RULE-11 | error | Identifier not in scope | Expression |
| RULE-12 | error | Type mismatch in expression | Expression |
| RULE-13 | error | Collection operation missing explicit lambda parameter | Expression |
| RULE-14 | error | Cannot compare inline enums from different fields | Expression |
| RULE-15 | error | Discriminator variant names must be PascalCase | Structural |
| RULE-16 | error | Discriminator variant has no matching variant declaration | Sum Type |
| RULE-17 | error | Variant not listed in base entity discriminator | Sum Type |
| RULE-18 | error | Variant field accessed without type guard | Sum Type |
| RULE-19 | error | Must use variant name for creation when discriminator exists | Sum Type |
| RULE-20 | error | Enumeration values must be non-empty | Structural |
| RULE-21 | error | Variant declaration requires name and base_entity | Structural |
| RULE-22 | error | Given binding type not declared | Reference |
| RULE-23 | error | Duplicate given binding name | Uniqueness |
| RULE-24 | error | Given binding requires name and type | Structural |
| RULE-25 | error | Config parameter requires name, type, and default_value | Structural |
| RULE-26 | error | Duplicate config parameter name | Uniqueness |
| RULE-27 | error | Config parameter referenced but not declared | Reference |
| RULE-28 | error | Surface facing type not declared | Reference |
| RULE-29 | error | Unreachable path in surface exposes | Surface |
| RULE-30 | error | Surface provides trigger not declared | Reference |
| RULE-31 | error | Surface related surface name not declared | Reference |
| RULE-32 | error | Unused binding in surface | Surface |
| RULE-33 | error | Invalid when condition reference in surface | Surface |
| RULE-34 | error | Cannot iterate over non-collection type | Surface |
| RULE-35 | error | Use declaration imports unresolvable type | Reference |

## All Warnings

| ID | Description |
|----|-------------|
| WARN-01 | External entity has no governing spec |
| WARN-02 | Open questions present |
| WARN-03 | Deferred spec has no location hint |
| WARN-04 | Unused entity or field |
| WARN-05 | Rule can never fire (contradictory requires) |
| WARN-06 | Temporal rule has no re-firing guard |
| WARN-07 | Surface exposes unused field |
| WARN-08 | Provides has impossible when condition |
| WARN-09 | Unused actor |
| WARN-10 | Sibling rule creates entity without duplicate guard |
| WARN-11 | Provides condition weaker than rule requires |
| WARN-12 | Overlapping preconditions on shared trigger |
| WARN-13 | Derived value references out-of-entity field |
| WARN-14 | Trivial actor identified_by condition |
| WARN-15 | All-conditional ensures with empty path |
| WARN-16 | Temporal trigger on optional field |
| WARN-17 | Raw entity type used when actors available |
| WARN-18 | transitions_to fires on creation value |
| WARN-19 | Multiple identical inline enums suggest named enum |

See [warnings.md](warnings.md) for full details on each warning.
