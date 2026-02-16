# Feature Specification: Allium Validator

**Created**: 2026-02-15
**Status**: Draft
**Input**: ALLIUM_VALIDATOR_PLAN.md — Three-layer JSON Schema validation for the Allium behavioral specification language

---

## User Stories & Acceptance Criteria

### User Story 1 — JSON Schema Structural Validation (Priority: P0)

A spec author creates a `.allium.json` file (either manually or via LLM conversion from `.allium`) and needs immediate feedback on whether the document is structurally well-formed. The JSON Schema layer catches ~40% of errors — missing fields, wrong types, invalid enum values, malformed constructs — before any semantic analysis runs. This provides fast, deterministic rejection of broken files and establishes the foundation for all downstream validation.

**Why this priority**: Without schema validation, the semantic checker would need to handle arbitrary malformed input, vastly increasing complexity. Schema validation is the foundation that all other layers depend on. It also enforces 7 of the 35 formal rules via structural constraints alone.

**Independent Test**: Run `allium-check --schema-only file.allium.json` against valid and invalid files. Delivers value immediately — catches structural errors without needing any semantic check code.

**Acceptance Scenarios**:

1. **Given** a well-formed `.allium.json` matching all schema constraints, **When** schema-validated, **Then** validation passes with no schema errors.
2. **Given** a document missing the required `version` field, **When** validated, **Then** schema error reports missing required property "version".
3. **Given** an entity with a field lacking the `type` property, **When** validated, **Then** schema error reported (enforces Rule 2: fields have types).
4. **Given** a rule with an empty `ensures` array, **When** validated, **Then** schema error reported (enforces Rule 4: rules require trigger + non-empty ensures).
5. **Given** a trigger with an unrecognized `kind` value, **When** validated, **Then** schema error reported (enforces Rule 5: trigger types must be one of 7 valid kinds).
6. **Given** a FieldType with an invalid `kind` discriminator, **When** validated, **Then** schema error identifying the invalid kind.
7. **Given** discriminator variant names not in PascalCase (e.g., `"my_variant"`), **When** validated, **Then** schema error (enforces Rule 15: discriminators capitalized).
8. **Given** a variant declaration missing `base_entity`, **When** validated, **Then** schema error (enforces Rule 21: variant keyword requires name + base_entity).
9. **Given** a config parameter missing `default_value`, **When** validated, **Then** schema error (enforces Rule 25: config requires name, type, and default_value).
10. **Given** an expression node with unrecognized `kind`, **When** validated, **Then** schema error.
11. **Given** field names using PascalCase instead of snake_case, **When** validated, **Then** schema error for naming pattern violation.
12. **Given** entity names using snake_case instead of PascalCase, **When** validated, **Then** schema error for naming pattern violation.
13. **Given** an EnsuresClause with unrecognized `kind`, **When** validated, **Then** schema error.
14. **Given** each of the 14 schema definition files (common, field-types, expressions, entities, enumerations, rules, surfaces, actors, config, defaults, given, use-declarations, deferred, open-questions) defines valid constructs for its domain, **When** a valid construct of that type is submitted, **Then** it passes that file's schema.

---

### User Story 2 — CLI Binary Interface (Priority: P0)

A spec author or CI pipeline runs `allium-check` from the command line to validate one or more `.allium.json` files. The CLI provides clear exit codes (0 = clean, 1 = validation errors, 2 = input/parse errors), supports text and JSON output formats, and offers flags for controlling which checks run. The CLI is a single static Go binary with no runtime dependencies, usable in any environment.

**Why this priority**: The CLI is the delivery vehicle for all validation. Without it, neither humans nor CI nor LLM skills can invoke the validator. It must exist before any semantic rules have value.

**Independent Test**: Build the binary with `go build`, run it against a valid `.allium.json` file, confirm exit code 0 and clean output. Usable even before semantic rules are implemented (schema-only mode).

**Acceptance Scenarios**:

1. **Given** a valid `.allium.json` file, **When** `allium-check file.allium.json` is run, **Then** exit code is 0 and output shows "0 errors, 0 warnings".
2. **Given** a file with validation errors, **When** `allium-check file.allium.json` is run, **Then** exit code is 1 and each error is listed with rule ID, severity, message, and location.
3. **Given** a nonexistent file path, **When** `allium-check missing.json` is run, **Then** exit code is 2 and error message includes "file not found".
4. **Given** a file containing invalid JSON, **When** `allium-check bad.json` is run, **Then** exit code is 2 and error message includes "parse error".
5. **Given** `--format json` flag, **When** running against any file, **Then** output is valid JSON matching the report schema (file, schema_valid, errors, warnings, summary).
6. **Given** `--format text` flag or no format flag, **When** running, **Then** output is human-readable text with one issue per line.
7. **Given** `--quiet` flag with warnings present, **When** running, **Then** only errors are shown; warnings are suppressed from output.
8. **Given** `--strict` flag with warnings but no errors, **When** running, **Then** exit code is 1 (warnings treated as errors).
9. **Given** `--schema-only` flag, **When** running, **Then** only JSON Schema validation runs; semantic checks are skipped entirely.
10. **Given** `--rules 7-9` flag, **When** running, **Then** only semantic rules 7, 8, and 9 are checked (plus schema validation).
11. **Given** `--version` flag, **When** running, **Then** version string is printed and program exits immediately.
12. **Given** multiple file arguments, **When** running, **Then** each file is validated independently and results are reported per file.

---

### User Story 3 — Reference Resolution (Priority: P0)

A spec author writes a `.allium.json` that references entities, rules, actors, surfaces, config parameters, or imported types by name. The validator resolves every name reference against the declared symbols and reports any reference to an undeclared name. This catches the most common class of spec errors: typos and forgotten declarations.

**Why this priority**: Unresolved references make every downstream check unreliable. Reference resolution is the first semantic pass and the foundation for all others. It covers 8 of the 35 validation rules.

**Independent Test**: Create a minimal `.allium.json` with one entity and one rule referencing an undeclared entity. Run `allium-check` and confirm the undeclared reference is reported. Verify that fixing the reference makes the error disappear.

**Acceptance Scenarios**:

1. **Given** an `entity_ref` type referencing an entity name not in `entities`, `external_entities`, or `use_declarations`, **When** checked, **Then** error RULE-01 reports "Entity 'X' referenced but not declared" (Rule 1).
2. **Given** a relationship with `target_entity` not matching any declared entity, **When** checked, **Then** error RULE-03 reports the unresolved relationship target (Rule 3).
3. **Given** a `given` binding with a type reference to a nonexistent entity or value type, **When** checked, **Then** error RULE-22 reported (Rule 22).
4. **Given** an expression referencing a config parameter name not in `config`, **When** checked, **Then** error RULE-27 reported (Rule 27).
5. **Given** a surface `facing` clause with a type not matching any declared entity or actor, **When** checked, **Then** error RULE-28 reported (Rule 28).
6. **Given** a surface `provides` clause with a trigger name not matching any declared rule trigger, **When** checked, **Then** error RULE-30 reported (Rule 30).
7. **Given** a surface `related` clause with a `surface_name` not matching any declared surface, **When** checked, **Then** error RULE-31 reported (Rule 31).
8. **Given** a `use_declaration` importing a type that doesn't exist in the source specification, **When** checked, **Then** error RULE-35 reported (Rule 35).
9. **Given** all references resolve to declared names, **When** checked, **Then** no reference resolution errors are reported.

---

### User Story 4 — Uniqueness Enforcement (Priority: P0)

A spec author declares rules, given bindings, or config parameters. The validator ensures that names which must be unique within their scope are not duplicated, and that rules sharing a trigger name are parameter-compatible.

**Why this priority**: Duplicate declarations cause ambiguity that silently corrupts downstream analysis. This pass is simple, fast, and eliminates an entire class of bugs.

**Independent Test**: Create a `.allium.json` with two given bindings having the same name. Run `allium-check` and confirm the duplicate is reported.

**Acceptance Scenarios**:

1. **Given** two rules with the same trigger name but different parameter counts, **When** checked, **Then** error RULE-06 reports "Rules sharing trigger 'X' have incompatible parameters" (Rule 6).
2. **Given** two rules with the same trigger name and matching parameter signatures, **When** checked, **Then** no error for Rule 6.
3. **Given** two `given` bindings with the same name, **When** checked, **Then** error RULE-23 reports "Duplicate given binding name 'X'" (Rule 23).
4. **Given** two `config` parameters with the same name, **When** checked, **Then** error RULE-26 reports "Duplicate config parameter name 'X'" (Rule 26).
5. **Given** all names are unique within their scopes, **When** checked, **Then** no uniqueness errors.

---

### User Story 5 — State Machine Analysis (Priority: P1)

A spec author defines entities with enum-typed status fields and rules that transition between states. The validator builds a state machine graph from creation points and transition rules, then checks that all states are reachable, non-terminal states have outgoing transitions, and all assigned values are declared.

**Why this priority**: State machine bugs are the second most common spec error after reference typos. Unreachable states and dead ends indicate missing rules or impossible flows.

**Independent Test**: Create an entity with status `pending | active | completed | archived` where no rule transitions to `archived`. Run `allium-check` and confirm `archived` is reported as unreachable.

**Acceptance Scenarios**:

1. **Given** an entity with status values where some are not reachable from any creation point via transitions, **When** checked, **Then** error RULE-07 reports each unreachable value (Rule 7).
2. **Given** a non-terminal status value with no outgoing transition rule, **When** checked, **Then** error RULE-08 reports "Dead-end state 'X' has no outgoing transition" (Rule 8).
3. **Given** an ensures clause setting a status field to a value not declared in the enum, **When** checked, **Then** error RULE-09 reports "Undeclared status value 'X'" (Rule 9).
4. **Given** all status values are reachable with valid transitions and no dead ends, **When** checked, **Then** no state machine errors.
5. **Given** an entity with no enum-typed fields, **When** checked, **Then** state machine analysis is skipped for that entity.

---

### User Story 6 — Expression Validation (Priority: P1)

A spec author writes expressions in derived values, requires clauses, filters, and ensures. The validator checks that derived value dependencies are acyclic, all field access paths are in scope, types are compatible in comparisons and arithmetic, lambda parameters are explicit, and inline enum comparisons follow the rules.

**Why this priority**: Expression errors are subtle — cycles cause infinite loops, scope violations produce undefined behavior, and type mismatches silently corrupt logic. These are the hardest errors to catch by manual review.

**Independent Test**: Create two derived values that reference each other (cycle). Run `allium-check` and confirm the cycle is reported.

**Acceptance Scenarios**:

1. **Given** derived values A and B where A references B and B references A, **When** checked, **Then** error RULE-10 reports "Cycle detected in derived values: A -> B -> A" (Rule 10).
2. **Given** a `field_access` path whose root is not in scope (not a trigger binding, for-clause binding, let binding, given binding, or default), **When** checked, **Then** error RULE-11 reports "Identifier 'X' is not in scope" (Rule 11).
3. **Given** a comparison where left side is Integer and right side is String, **When** checked, **Then** error RULE-12 reports "Type mismatch in comparison" (Rule 12).
4. **Given** arithmetic on non-numeric, non-temporal types, **When** checked, **Then** error RULE-12 reported (Rule 12).
5. **Given** an `any` or `all` expression without an explicit `lambda_param`, **When** checked, **Then** error RULE-13 reported (Rule 13).
6. **Given** a comparison between two different inline enum fields, **When** checked, **Then** error RULE-14 reports "Cannot compare inline enums" (Rule 14).
7. **Given** a comparison between two named enums of the same type, **When** checked, **Then** no error (named enum comparison is valid).
8. **Given** all expressions are well-typed and in scope, **When** checked, **Then** no expression errors.

---

### User Story 7 — Sum Type Enforcement (Priority: P1)

A spec author defines entity discriminators (sum types) with variant declarations. The validator ensures that every discriminator variant has a corresponding variant declaration, every variant is listed in its base entity, variant-specific fields are only accessed within type guards, and entity creation uses variant names when discriminators exist.

**Why this priority**: Sum type errors produce specs that are structurally unsound — missing variants mean incomplete handling, unguarded access means runtime type errors, and wrong creation patterns bypass the type system.

**Independent Test**: Create an entity with discriminator `Branch | Leaf` but only declare a `Branch` variant. Run `allium-check` and confirm `Leaf` variant is reported missing.

**Acceptance Scenarios**:

1. **Given** a discriminator listing variant `X` but no `variant X : Entity` declaration exists, **When** checked, **Then** error RULE-16 reports "Discriminator variant 'X' has no matching variant declaration" (Rule 16).
2. **Given** a variant declaration `variant X : Entity` but `X` is not listed in Entity's discriminator, **When** checked, **Then** error RULE-17 reports "Variant 'X' not listed in base entity's discriminator" (Rule 17).
3. **Given** access to a variant-specific field outside a type guard, **When** checked, **Then** error RULE-18 reports "Variant field 'X' accessed without type guard" (Rule 18).
4. **Given** access to a variant-specific field inside a type guard (requires clause or if branch checking discriminator), **When** checked, **Then** no error for that access.
5. **Given** entity creation via `.created()` using the base entity name when a discriminator exists, **When** checked, **Then** error RULE-19 reports "Must use variant name for creation when discriminator exists" (Rule 19).
6. **Given** correct sum type usage (all variants declared, listed, guarded, and created properly), **When** checked, **Then** no sum type errors.

---

### User Story 8 — Surface Semantic Validation (Priority: P1)

A spec author defines surfaces (boundary contracts between parties) with facing, context, exposes, provides, related, and when conditions. The validator ensures that all exposed field paths are reachable, bindings are used, when conditions reference valid fields, and iterations target collections.

**Why this priority**: Surfaces are the primary interface between the spec and its users. Incorrect surfaces mean the user-facing contract doesn't match the domain model.

**Independent Test**: Create a surface that exposes a field path not reachable from facing or context. Run `allium-check` and confirm the unreachable path is reported.

**Acceptance Scenarios**:

1. **Given** an `exposes` entry with a field path not reachable from facing, context, or let bindings, **When** checked, **Then** error RULE-29 reports "Unreachable path in exposes" (Rule 29).
2. **Given** a facing or context binding that is never referenced anywhere in the surface body, **When** checked, **Then** error RULE-32 reports "Unused binding 'X'" (Rule 32).
3. **Given** a `when` condition referencing a field not reachable from party or context bindings, **When** checked, **Then** error RULE-33 reported (Rule 33).
4. **Given** a `for` iteration targeting a non-collection-typed field, **When** checked, **Then** error RULE-34 reports "Cannot iterate over non-collection type" (Rule 34).
5. **Given** a valid surface with all paths reachable, bindings used, and conditions valid, **When** checked, **Then** no surface errors.

---

### User Story 9 — Warning Detection (Priority: P2)

A spec author produces a valid spec (no errors) but the validator detects potential issues — unused declarations, unguarded temporal rules, open questions, and other patterns that often indicate bugs or incomplete specs. Each warning has a unique ID and clear message.

**Why this priority**: Warnings are not blockers but catch real issues. They're less urgent than error rules but essential for spec quality. All 19 warning checks require the full symbol table and pass infrastructure to be in place.

**Independent Test**: Create a spec with an unused entity (never referenced by any rule, surface, or relationship). Run `allium-check` and confirm the unused entity warning appears.

**Acceptance Scenarios**:

1. **Given** an external entity not referenced by any `use` declaration's imported types, **When** checked, **Then** warning WARN-01 "External entity 'X' has no governing spec" (WARN-01).
2. **Given** any entry in `open_questions`, **When** checked, **Then** warning WARN-02 "Open questions present: N unresolved" (WARN-02).
3. **Given** a deferred spec with null or empty `location_hint`, **When** checked, **Then** warning WARN-03 "Deferred spec 'X' has no location hint" (WARN-03).
4. **Given** an entity or field not referenced by any rule, surface, relationship, or other entity, **When** checked, **Then** warning WARN-04 "Unused entity/field 'X'" (WARN-04).
5. **Given** a rule with contradictory requires (e.g., `status = A and status = B`), **When** checked, **Then** warning WARN-05 "Rule 'X' can never fire" (WARN-05).
6. **Given** a temporal trigger without a requires clause that prevents re-firing, **When** checked, **Then** warning WARN-06 "Temporal rule 'X' has no re-firing guard" (WARN-06).
7. **Given** a surface exposes a field not used by any rule, **When** checked, **Then** warning WARN-07 "Surface exposes unused field" (WARN-07).
8. **Given** a provides clause with a `when` condition that is always false, **When** checked, **Then** warning WARN-08 "Provides 'X' has impossible when condition" (WARN-08).
9. **Given** an actor declaration not referenced in any surface `facing` clause, **When** checked, **Then** warning WARN-09 "Unused actor 'X'" (WARN-09).
10. **Given** a rule that creates a child entity for a parent without guarding against duplicates, **When** checked, **Then** warning WARN-10 "Sibling rule 'X' creates entity without duplicate guard" (WARN-10).
11. **Given** a surface provides `when` condition strictly weaker than the corresponding rule's requires, **When** checked, **Then** warning WARN-11 "Provides condition weaker than rule requires" (WARN-11).
12. **Given** two rules with the same trigger whose requires clauses could be simultaneously true, **When** checked, **Then** warning WARN-12 "Overlapping preconditions on trigger 'X'" (WARN-12).
13. **Given** a parameterised derived value referencing fields outside its owning entity, **When** checked, **Then** warning WARN-13 "Derived value 'X' references out-of-entity field" (WARN-13).
14. **Given** an actor `identified_by` condition that always evaluates to true or always false, **When** checked, **Then** warning WARN-14 "Trivial actor identified_by condition" (WARN-14).
15. **Given** all ensures clauses are conditional and at least one path produces no effects, **When** checked, **Then** warning WARN-15 "All-conditional ensures with empty path" (WARN-15).
16. **Given** a temporal trigger on an optional field (type `T?`), **When** checked, **Then** warning WARN-16 "Temporal trigger on optional field — won't fire when absent" (WARN-16).
17. **Given** a surface using a raw entity type in `facing` when actors exist for that entity, **When** checked, **Then** warning WARN-17 "Raw entity type used when actors available" (WARN-17).
18. **Given** a `transitions_to` trigger on a status value that entities can be created with, **When** checked, **Then** warning WARN-18 "transitions_to fires on creation value" (WARN-18).
19. **Given** the same entity has multiple fields with identical inline enum literal sets, **When** checked, **Then** warning WARN-19 "Multiple identical inline enums — consider a named enum" (WARN-19).
20. **Given** a clean spec with no warning-triggering patterns, **When** checked, **Then** no warnings are reported.

---

### User Story 10 — Validate Skill with Guidance Checks (Priority: P2)

A spec author invokes the `validate` skill on a `.allium` file (or `.allium.json`). The skill converts `.allium` to `.allium.json` (if needed), runs `allium-check`, interprets the results with human-readable explanations and fix suggestions, and then runs LLM guidance-level checks that require reasoning beyond deterministic rules — naming quality, spec completeness, and domain appropriateness.

**Why this priority**: The skill is the primary interface for LLM-driven workflows. It bridges the deterministic CLI with the LLM's ability to suggest fixes and assess subjective quality. However, it depends on the CLI being complete.

**Independent Test**: Provide a `.allium` file with a known error. Invoke the validate skill. Confirm it converts to JSON, runs the CLI, and presents the error with a suggested fix.

**Acceptance Scenarios**:

1. **Given** a `.allium` file, **When** the validate skill is invoked, **Then** it generates a `.allium.json` alongside, runs `allium-check`, and reports the results.
2. **Given** `allium-check` returns errors, **When** the skill processes output, **Then** each error is presented with a human-readable explanation and a suggested fix (e.g., "Entity 'FooBar' is referenced but not declared — did you mean 'Foobar'?").
3. **Given** `allium-check` returns clean (no errors, no warnings), **When** the skill runs guidance checks, **Then** it reports any naming quality issues (vague rule names, non-descriptive entity names).
4. **Given** a spec with vague rule names (e.g., "DoThing", "HandleStuff"), **When** guidance checks run, **Then** the skill flags each vague name with a suggestion for improvement.
5. **Given** a spec missing common domain patterns (e.g., no temporal rules for time-sensitive entities), **When** guidance checks run, **Then** the skill suggests areas that may need additional specification.
6. **Given** the `allium-check` binary is not found in PATH, **When** the validate skill is invoked, **Then** a clear error message explains how to install or build the binary.

---

### User Story 11 — Distill and Elicit Skill Integration (Priority: P2)

A spec author uses the `distill` skill (extract spec from code) or `elicit` skill (build spec through conversation). After producing a `.allium` file, these skills automatically generate a `.allium.json` alongside, run validation, and either fix errors before presenting results (distill) or use errors as conversation prompts (elicit).

**Why this priority**: Integration closes the feedback loop — specs are validated as they're created, not after. Depends on both the CLI and the validate skill being available.

**Independent Test**: Run the distill skill against a code sample. Confirm that a `.allium.json` file is generated alongside the `.allium` file and that validation results are reported.

**Acceptance Scenarios**:

1. **Given** the distill skill produces a `.allium` file (Step 7 complete), **When** Step 8 runs, **Then** a parallel `.allium.json` is generated and `allium-check` is invoked.
2. **Given** validation errors are found during distill Step 8, **When** errors are detected, **Then** the skill fixes the errors before presenting the final spec to the user.
3. **Given** the elicit skill produces a `.allium` file after session output, **When** the output is generated, **Then** a `.allium.json` is generated and validated.
4. **Given** validation errors during elicit Phase 4 (Refinement), **When** errors are found, **Then** the errors are used as conversation prompts (e.g., "Entity X has unreachable status value 'archived' — is that intentional?").

---

### User Story 12 — Reference Example (Priority: P0)

A developer building the validator needs a reference `.allium.json` file that exercises all major constructs (entities, relationships, projections, derived values, all 7 trigger types, all 8 ensures clause types, expressions, surfaces, actors, config, given, variants, enumerations). The `password-auth` pattern from the Allium patterns library serves as this reference. It must pass all validations cleanly.

**Why this priority**: The reference example is the integration test fixture that validates the entire pipeline end-to-end. Without it, there's no way to confirm the validator works as a whole. It must be created before or alongside the schema definitions.

**Independent Test**: Run `allium-check schemas/v1/examples/password-auth.allium.json` and confirm exit code 0, zero errors, zero warnings.

**Acceptance Scenarios**:

1. **Given** the password-auth pattern from the patterns library, **When** converted to `.allium.json` following the JSON AST spec, **Then** all construct types are represented (entities, external entities, value types, enumerations, rules with diverse triggers, surfaces, actors, config, given, defaults, variants if applicable).
2. **Given** `password-auth.allium.json` exists, **When** `allium-check` runs against it, **Then** exit code is 0 with 0 errors and 0 warnings.
3. **Given** intentionally broken variants of `password-auth.allium.json` (undeclared references, duplicate names, unreachable states, cycles, unguarded variant access), **When** `allium-check` runs, **Then** each broken variant produces the correct error for the specific rule being violated.

---

### User Story 13 — Validation Documentation (Priority: P3)

A spec author encountering a validation error or warning needs documentation explaining what the rule checks, why it matters, and how to fix violations. The documentation maps each rule ID to its check, provides examples of violations and fixes, and serves as the authoritative reference for the validation system.

**Why this priority**: Documentation is important but not blocking — the CLI's error messages provide immediate guidance. Formal documentation can follow the implementation.

**Independent Test**: Open `VALIDATION-RULES.md` and confirm every rule ID (RULE-01 through RULE-35) and warning ID (WARN-01 through WARN-19) appears with a description.

**Acceptance Scenarios**:

1. **Given** `VALIDATION-RULES.md` exists, **When** read, **Then** every rule (1-35) and warning (1-19) has an entry with ID, description, severity, and implementation mapping.
2. **Given** each of the 7 rule group documents (structural.md, state-machine.md, expression.md, sum-type.md, given.md, config.md, surface.md) exists, **When** read, **Then** each rule in the group has an explanation, an example violation, and an example fix.
3. **Given** `warnings.md` exists, **When** read, **Then** each warning has an ID, description, example trigger, and suggested resolution.

---

## Edge Cases

- What happens when `.allium.json` is empty (`{}`)? Expected: schema error for all required top-level fields (version, file, entities, rules, etc.).
- What happens when `.allium.json` is valid JSON but not an object (e.g., `[]` or `"string"`)? Expected: schema error for root type mismatch.
- What happens when an entity references itself in a relationship? Expected: valid — self-referential relationships are allowed (e.g., `parent: Node with parent = this`).
- What happens when a rule has an empty `requires` array? Expected: valid — rules with no preconditions are allowed.
- What happens when a surface has no `exposes` or `provides`? Expected: valid — a view-only surface with only facing/context is allowed (though likely a WARN-07 candidate).
- What happens when a variant has zero variant-specific fields? Expected: valid — a variant may differ only by discriminator value.
- What happens when deeply nested expressions (10+ levels of boolean/arithmetic) are validated? Expected: works correctly — no stack overflow for small spec sizes.
- What happens when a given binding has the same name as an entity? Expected: the binding shadows the entity name in its scope; this is valid but could trigger a future warning.
- What happens when all ensures clauses in a rule are inside conditionals with no else? Expected: WARN-15 triggered (all-conditional ensures with empty path).
- What happens when a temporal trigger references `now` on an entity with no Timestamp fields? Expected: schema passes (expression is valid), but the comparison types should trigger RULE-12 (type mismatch).
- What happens when `--rules` flag specifies rule IDs that don't exist (e.g., `--rules 99`)? Expected: CLI ignores unknown rule IDs and validates only recognized ones.
- What happens when the same file is passed multiple times as arguments? Expected: each instance is validated independently; results may be duplicated.
- What happens when a field's type is `optional` wrapping another `optional` (e.g., `T??`)? Expected: schema allows it (structurally valid), but it may be flagged as a potential warning.
- What happens when the `--strict` and `--quiet` flags are used together? Expected: `--quiet` suppresses warning output, but `--strict` still causes exit code 1 if warnings exist (the count is still checked).
- What happens when a config parameter's `default_value` type doesn't match the declared type? Expected: this should be caught by semantic validation (type checking), not schema alone.

---

## BDD Scenarios

### Feature: JSON Schema Structural Validation

#### Background

- **Given** the `allium-check` binary is built and available
- **And** the JSON Schema files are embedded in the binary

---

#### Scenario: Complete valid document passes schema validation

**Traces to**: User Story 1, Acceptance Scenario 1
**Category**: Happy Path

- **Given** a `.allium.json` file that conforms to all schema constraints
- **When** `allium-check --schema-only file.allium.json` is run
- **Then** exit code is 0
- **And** output contains no schema errors

---

#### Scenario Outline: Schema rejects documents with missing required fields

**Traces to**: User Story 1, Acceptance Scenario 2
**Category**: Error Path

- **Given** a `.allium.json` file missing the required field `<field>`
- **When** `allium-check --schema-only file.allium.json` is run
- **Then** exit code is 1
- **And** output contains a schema error referencing `<field>`

**Examples**:

| field | description |
|-------|-------------|
| `version` | Top-level version number |
| `file` | Source file name |
| `entities` | Entities array |
| `rules` | Rules array |

---

#### Scenario Outline: Schema enforces structural rules

**Traces to**: User Story 1, Acceptance Scenarios 3-9
**Category**: Error Path

- **Given** a `.allium.json` file with `<violation>`
- **When** `allium-check --schema-only file.allium.json` is run
- **Then** exit code is 1
- **And** output contains a schema error for `<rule_desc>`

**Examples**:

| violation | rule_desc | rule_id |
|-----------|-----------|---------|
| Entity field missing `type` property | Field requires name + type (Rule 2) | Rule 2 |
| Rule with empty `ensures` array | ensures requires minItems: 1 (Rule 4) | Rule 4 |
| Trigger with `kind: "invalid"` | Trigger must be one of 7 kinds (Rule 5) | Rule 5 |
| Discriminator variant in snake_case | Variants must be PascalCase (Rule 15) | Rule 15 |
| Variant missing `base_entity` | Variant requires name + base_entity (Rule 21) | Rule 21 |
| Config parameter missing `default_value` | Config requires name, type, default (Rule 25) | Rule 25 |

---

#### Scenario Outline: Schema enforces naming patterns

**Traces to**: User Story 1, Acceptance Scenarios 11-12
**Category**: Error Path

- **Given** a `.allium.json` where `<element>` uses `<wrong_case>` instead of `<expected_case>`
- **When** `allium-check --schema-only file.allium.json` is run
- **Then** exit code is 1
- **And** output contains a naming pattern error

**Examples**:

| element | wrong_case | expected_case |
|---------|-----------|---------------|
| Entity name | `my_entity` | PascalCase |
| Field name | `MyField` | snake_case |
| Rule name | `handle_reset` | PascalCase |
| Config parameter | `MaxRetries` | snake_case |
| Enum literal | `Active` | snake_case |

---

#### Scenario Outline: Each schema definition file validates its construct type

**Traces to**: User Story 1, Acceptance Scenario 14
**Category**: Happy Path

- **Given** a `.allium.json` containing a valid `<construct>` definition
- **When** schema-validated
- **Then** the `<schema_file>` definition accepts the construct without errors

**Examples**:

| construct | schema_file |
|-----------|-------------|
| Metadata with scope, includes, excludes | common.json |
| FieldType with all 7 kinds | field-types.json |
| Expression with representative kinds | expressions.json |
| Entity with fields, relationships, projections, derived values | entities.json |
| Named enumeration | enumerations.json |
| Rule with trigger and ensures | rules.json |
| Surface with facing, context, exposes, provides, related | surfaces.json |
| Actor with within and identified_by | actors.json |
| Config parameter with name, type, default_value | config.json |
| Default declaration | defaults.json |
| Given block with bindings | given.json |
| Use declaration with imports | use-declarations.json |
| Deferred spec with location_hint | deferred.json |
| Open question | open-questions.json |

---

### Feature: CLI Binary Interface

#### Scenario: CLI reports clean validation

**Traces to**: User Story 2, Acceptance Scenario 1
**Category**: Happy Path

- **Given** a valid `.allium.json` file with no errors or warnings
- **When** `allium-check file.allium.json` is run
- **Then** exit code is 0
- **And** text output includes "0 errors" and "0 warnings"

---

#### Scenario: CLI reports validation errors with details

**Traces to**: User Story 2, Acceptance Scenario 2
**Category**: Happy Path

- **Given** a `.allium.json` file with a reference to undeclared entity "FooBar"
- **When** `allium-check file.allium.json` is run
- **Then** exit code is 1
- **And** output includes rule ID "RULE-01", severity "error", and entity name "FooBar"

---

#### Scenario Outline: CLI handles input errors

**Traces to**: User Story 2, Acceptance Scenarios 3-4
**Category**: Error Path

- **Given** `<input_condition>`
- **When** `allium-check <input>` is run
- **Then** exit code is 2
- **And** output includes `<error_message>`

**Examples**:

| input_condition | input | error_message |
|-----------------|-------|---------------|
| File does not exist | `missing.json` | "file not found" |
| File contains invalid JSON | `bad.json` | "parse error" |
| File is empty (0 bytes) | `empty.json` | "parse error" |

---

#### Scenario: CLI outputs JSON format

**Traces to**: User Story 2, Acceptance Scenario 5
**Category**: Happy Path

- **Given** any `.allium.json` file
- **When** `allium-check --format json file.allium.json` is run
- **Then** output is valid JSON
- **And** JSON contains keys: "file", "schema_valid", "errors", "warnings", "summary"

---

#### Scenario Outline: CLI flag behavior

**Traces to**: User Story 2, Acceptance Scenarios 7-11
**Category**: Alternate Path

- **Given** a `.allium.json` file with `<file_state>`
- **When** `allium-check <flags> file.allium.json` is run
- **Then** `<expected_behavior>`

**Examples**:

| file_state | flags | expected_behavior |
|------------|-------|-------------------|
| 1 warning, 0 errors | `--quiet` | Exit 0, warning not shown in output |
| 1 warning, 0 errors | `--strict` | Exit 1, warning shown as error |
| Schema error + semantic error | `--schema-only` | Only schema error reported, semantic error not checked |
| Errors on rules 7 and 10 | `--rules 7-9` | Only rule 7 error reported, rule 10 skipped |
| Any file | `--version` | Version string printed, no validation performed |

---

#### Scenario: CLI validates multiple files independently

**Traces to**: User Story 2, Acceptance Scenario 12
**Category**: Alternate Path

- **Given** two `.allium.json` files: `valid.allium.json` (clean) and `broken.allium.json` (has errors)
- **When** `allium-check valid.allium.json broken.allium.json` is run
- **Then** exit code is 1 (at least one file has errors)
- **And** output shows results for each file separately
- **And** `valid.allium.json` section shows 0 errors
- **And** `broken.allium.json` section shows the specific errors

---

### Feature: Reference Resolution

#### Scenario Outline: Undeclared references are detected

**Traces to**: User Story 3, Acceptance Scenarios 1-8
**Category**: Error Path

- **Given** a `.allium.json` where `<location>` references `<name>` which is not declared
- **When** `allium-check file.allium.json` is run
- **Then** error `<rule_id>` is reported with message containing `<name>`

**Examples**:

| location | name | rule_id | rule |
|----------|------|---------|------|
| Entity field type `entity_ref` | `FooBar` | RULE-01 | Entity reference resolution |
| Relationship `target_entity` | `NonExistent` | RULE-03 | Relationship target resolution |
| Given binding type | `UnknownType` | RULE-22 | Given type resolution |
| Expression config reference | `missing_param` | RULE-27 | Config reference resolution |
| Surface `facing` type | `UnknownActor` | RULE-28 | Surface facing resolution |
| Surface `provides` trigger | `NonExistentRule` | RULE-30 | Surface provides resolution |
| Surface `related` surface_name | `MissingSurface` | RULE-31 | Surface related resolution |
| Use declaration import | `ExternalType` | RULE-35 | Use declaration resolution |

---

#### Scenario: All references resolve successfully

**Traces to**: User Story 3, Acceptance Scenario 9
**Category**: Happy Path

- **Given** a `.allium.json` where every entity_ref, relationship target, given type, config reference, surface facing, provides, related, and use declaration import resolves to a declared name
- **When** `allium-check file.allium.json` is run
- **Then** no reference resolution errors are reported

---

### Feature: Uniqueness Enforcement

#### Scenario: Incompatible trigger parameter signatures detected

**Traces to**: User Story 4, Acceptance Scenario 1
**Category**: Error Path

- **Given** two rules both triggered by `UserSubmitsForm` but with different parameter counts (one has 2 parameters, one has 3)
- **When** `allium-check file.allium.json` is run
- **Then** error RULE-06 reports incompatible parameters for trigger `UserSubmitsForm`

---

#### Scenario: Compatible trigger signatures pass

**Traces to**: User Story 4, Acceptance Scenario 2
**Category**: Happy Path

- **Given** two rules both triggered by `UserSubmitsForm` with matching parameter counts and positional types
- **When** `allium-check file.allium.json` is run
- **Then** no RULE-06 error is reported

---

#### Scenario Outline: Duplicate names in scoped declarations detected

**Traces to**: User Story 4, Acceptance Scenarios 3-4
**Category**: Error Path

- **Given** two `<declaration_type>` entries both named `<name>`
- **When** `allium-check file.allium.json` is run
- **Then** error `<rule_id>` reports duplicate name `<name>`

**Examples**:

| declaration_type | name | rule_id |
|-----------------|------|---------|
| Given bindings | `current_user` | RULE-23 |
| Config parameters | `max_retries` | RULE-26 |

---

### Feature: State Machine Analysis

#### Scenario: Unreachable status value detected

**Traces to**: User Story 5, Acceptance Scenario 1
**Category**: Error Path

- **Given** an entity `Order` with status `pending | processing | completed | archived`
- **And** creation sets status to `pending`
- **And** rules transition `pending -> processing -> completed` but nothing transitions to `archived`
- **When** `allium-check file.allium.json` is run
- **Then** error RULE-07 reports "Unreachable status value 'archived' on Order"

---

#### Scenario: Dead-end non-terminal state detected

**Traces to**: User Story 5, Acceptance Scenario 2
**Category**: Error Path

- **Given** an entity `Task` with status `open | in_progress | blocked | done`
- **And** rules transition `open -> in_progress -> done` and `open -> blocked`
- **And** no rule transitions from `blocked` to any other state
- **And** `blocked` is not semantically terminal
- **When** `allium-check file.allium.json` is run
- **Then** error RULE-08 reports "Dead-end state 'blocked' has no outgoing transition"

---

#### Scenario: Undeclared status value in assignment

**Traces to**: User Story 5, Acceptance Scenario 3
**Category**: Error Path

- **Given** an entity with status `pending | active | completed`
- **And** a rule's ensures sets status to `cancelled` (not in the enum)
- **When** `allium-check file.allium.json` is run
- **Then** error RULE-09 reports "Undeclared status value 'cancelled'"

---

#### Scenario: Valid state machine passes

**Traces to**: User Story 5, Acceptance Scenario 4
**Category**: Happy Path

- **Given** all status values are reachable from creation points via transitions
- **And** all non-terminal values have at least one outgoing transition
- **And** all assigned values are declared in the enum
- **When** `allium-check file.allium.json` is run
- **Then** no state machine errors are reported

---

### Feature: Expression Validation

#### Scenario: Circular derived value dependency detected

**Traces to**: User Story 6, Acceptance Scenario 1
**Category**: Error Path

- **Given** entity `Order` with derived value `total` referencing `tax` and derived value `tax` referencing `total`
- **When** `allium-check file.allium.json` is run
- **Then** error RULE-10 reports "Cycle detected in derived values: total -> tax -> total"

---

#### Scenario: Out-of-scope field access detected

**Traces to**: User Story 6, Acceptance Scenario 2
**Category**: Error Path

- **Given** a rule whose requires clause accesses `unknown_binding.status`
- **And** `unknown_binding` is not a trigger binding, for-clause binding, let binding, given binding, or default
- **When** `allium-check file.allium.json` is run
- **Then** error RULE-11 reports "Identifier 'unknown_binding' is not in scope"

---

#### Scenario Outline: Type mismatches in expressions

**Traces to**: User Story 6, Acceptance Scenarios 3-4
**Category**: Error Path

- **Given** an expression with `<operation>` where `<type_issue>`
- **When** `allium-check file.allium.json` is run
- **Then** error RULE-12 reports `<message>`

**Examples**:

| operation | type_issue | message |
|-----------|-----------|---------|
| Comparison `=` | left is Integer, right is String | "Type mismatch in comparison: Integer vs String" |
| Arithmetic `+` | left is Boolean, right is Integer | "Arithmetic on non-numeric type: Boolean" |
| Comparison `<` | left is Timestamp, right is Integer | "Type mismatch in comparison: Timestamp vs Integer" |
| Arithmetic `-` | left is Timestamp, right is Duration | No error (Timestamp - Duration is valid) |

---

#### Scenario: Missing lambda parameter in collection operation

**Traces to**: User Story 6, Acceptance Scenario 5
**Category**: Error Path

- **Given** an `any` expression without an explicit `lambda_param`
- **When** `allium-check file.allium.json` is run
- **Then** error RULE-13 reports "Collection operation requires explicit lambda parameter"

---

#### Scenario: Inline enum cross-comparison rejected

**Traces to**: User Story 6, Acceptance Scenario 6
**Category**: Error Path

- **Given** a comparison between `order.status` (inline enum `pending | active`) and `task.state` (inline enum `open | closed`)
- **When** `allium-check file.allium.json` is run
- **Then** error RULE-14 reports "Cannot compare inline enums from different fields"

---

#### Scenario: Named enum comparison accepted

**Traces to**: User Story 6, Acceptance Scenario 7
**Category**: Happy Path

- **Given** a comparison between two fields both typed as named enum `Priority`
- **When** `allium-check file.allium.json` is run
- **Then** no RULE-14 error is reported

---

### Feature: Sum Type Enforcement

#### Scenario: Missing variant declaration for discriminator

**Traces to**: User Story 7, Acceptance Scenario 1
**Category**: Error Path

- **Given** entity `Node` with discriminator `Branch | Leaf`
- **And** only `variant Branch : Node` is declared (no `Leaf` variant)
- **When** `allium-check file.allium.json` is run
- **Then** error RULE-16 reports "Discriminator variant 'Leaf' has no matching variant declaration"

---

#### Scenario: Variant not listed in base entity discriminator

**Traces to**: User Story 7, Acceptance Scenario 2
**Category**: Error Path

- **Given** `variant Stem : Node` is declared
- **And** entity `Node` discriminator is `Branch | Leaf` (no `Stem`)
- **When** `allium-check file.allium.json` is run
- **Then** error RULE-17 reports "Variant 'Stem' not listed in Node's discriminator"

---

#### Scenario: Unguarded variant field access rejected

**Traces to**: User Story 7, Acceptance Scenario 3
**Category**: Error Path

- **Given** a rule accesses `node.children` (a Branch-specific field)
- **And** no type guard narrows `node` to `Branch` in the enclosing scope
- **When** `allium-check file.allium.json` is run
- **Then** error RULE-18 reports "Variant field 'children' accessed without type guard"

---

#### Scenario: Guarded variant field access accepted

**Traces to**: User Story 7, Acceptance Scenario 4
**Category**: Happy Path

- **Given** a rule with `requires: node.kind = Branch`
- **And** within that scope, the rule accesses `node.children`
- **When** `allium-check file.allium.json` is run
- **Then** no RULE-18 error for that access

---

#### Scenario: Entity creation must use variant name

**Traces to**: User Story 7, Acceptance Scenario 5
**Category**: Error Path

- **Given** entity `Node` has a discriminator `Branch | Leaf`
- **And** a rule's ensures creates `Node.created(...)` instead of `Branch.created(...)`
- **When** `allium-check file.allium.json` is run
- **Then** error RULE-19 reports "Must use variant name for creation when discriminator exists"

---

### Feature: Surface Semantic Validation

#### Scenario: Unreachable exposes path detected

**Traces to**: User Story 8, Acceptance Scenario 1
**Category**: Error Path

- **Given** a surface with `facing: viewer: User` and `context: order: Order`
- **And** an `exposes` entry referencing `product.name` (not reachable from `viewer` or `order`)
- **When** `allium-check file.allium.json` is run
- **Then** error RULE-29 reports "Unreachable path 'product.name' in exposes"

---

#### Scenario: Unused binding detected

**Traces to**: User Story 8, Acceptance Scenario 2
**Category**: Error Path

- **Given** a surface with `facing: viewer: User`
- **And** `viewer` is never referenced in exposes, provides, related, or let bindings
- **When** `allium-check file.allium.json` is run
- **Then** error RULE-32 reports "Unused binding 'viewer'"

---

#### Scenario: Invalid when condition reference

**Traces to**: User Story 8, Acceptance Scenario 3
**Category**: Error Path

- **Given** a surface provides clause with `when_condition` referencing `unknown.field`
- **And** `unknown` is not reachable from facing or context
- **When** `allium-check file.allium.json` is run
- **Then** error RULE-33 is reported

---

#### Scenario: Non-collection iteration rejected

**Traces to**: User Story 8, Acceptance Scenario 4
**Category**: Error Path

- **Given** a surface with a for-iteration over a field typed as `String` (not a collection)
- **When** `allium-check file.allium.json` is run
- **Then** error RULE-34 reports "Cannot iterate over non-collection type"

---

#### Scenario: Valid surface passes all checks

**Traces to**: User Story 8, Acceptance Scenario 5
**Category**: Happy Path

- **Given** a surface where all exposes paths are reachable, all bindings used, all when conditions valid, and all iterations target collections
- **When** `allium-check file.allium.json` is run
- **Then** no surface errors are reported

---

### Feature: Warning Detection

#### Scenario Outline: Structural warnings detected

**Traces to**: User Story 9, Acceptance Scenarios 1-4
**Category**: Edge Case

- **Given** a valid `.allium.json` containing `<pattern>`
- **When** `allium-check file.allium.json` is run
- **Then** warning `<warn_id>` is reported with message `<message>`
- **And** exit code is 0 (warnings are not errors by default)

**Examples**:

| pattern | warn_id | message |
|---------|---------|---------|
| External entity not in any use declaration | WARN-01 | "External entity 'X' has no governing spec" |
| Non-empty open_questions array | WARN-02 | "Open questions present: N unresolved" |
| Deferred spec with null location_hint | WARN-03 | "Deferred spec 'X' has no location hint" |
| Entity never referenced by any rule or surface | WARN-04 | "Unused entity 'X'" |

---

#### Scenario Outline: Rule quality warnings detected

**Traces to**: User Story 9, Acceptance Scenarios 5-6, 10, 12, 15, 18
**Category**: Edge Case

- **Given** a valid `.allium.json` containing `<pattern>`
- **When** `allium-check file.allium.json` is run
- **Then** warning `<warn_id>` is reported

**Examples**:

| pattern | warn_id | description |
|---------|---------|-------------|
| Rule with contradictory requires (`status = A and status = B`) | WARN-05 | Rule can never fire |
| Temporal trigger without re-firing guard | WARN-06 | Temporal rule without guard |
| Rule creates child without duplicate guard | WARN-10 | Sibling creation guard |
| Two rules with overlapping requires on same trigger | WARN-12 | Overlapping preconditions |
| All ensures conditional with at least one empty path | WARN-15 | Empty conditional path |
| transitions_to on a creation value | WARN-18 | Fires on creation |

---

#### Scenario Outline: Surface and actor warnings detected

**Traces to**: User Story 9, Acceptance Scenarios 7-9, 11, 17
**Category**: Edge Case

- **Given** a valid `.allium.json` containing `<pattern>`
- **When** `allium-check file.allium.json` is run
- **Then** warning `<warn_id>` is reported

**Examples**:

| pattern | warn_id | description |
|---------|---------|-------------|
| Surface exposes field used by no rule | WARN-07 | Unused exposed field |
| Provides with always-false when condition | WARN-08 | Impossible provides |
| Actor not referenced in any surface facing | WARN-09 | Unused actor |
| Provides when condition weaker than rule requires | WARN-11 | Weak provides guard |
| Surface facing uses raw entity when actors exist | WARN-17 | Raw entity with actors |

---

#### Scenario Outline: Expression and type warnings detected

**Traces to**: User Story 9, Acceptance Scenarios 13-14, 16, 19
**Category**: Edge Case

- **Given** a valid `.allium.json` containing `<pattern>`
- **When** `allium-check file.allium.json` is run
- **Then** warning `<warn_id>` is reported

**Examples**:

| pattern | warn_id | description |
|---------|---------|-------------|
| Parameterised derived value referencing out-of-entity field | WARN-13 | Derived scope violation |
| Actor identified_by always true/false | WARN-14 | Trivial actor condition |
| Temporal trigger on optional field (T?) | WARN-16 | Optional temporal field |
| Multiple fields with identical inline enum literals | WARN-19 | Duplicate inline enums |

---

#### Scenario: Clean spec produces no warnings

**Traces to**: User Story 9, Acceptance Scenario 20
**Category**: Happy Path

- **Given** a `.allium.json` with no warning-triggering patterns
- **When** `allium-check file.allium.json` is run
- **Then** output contains 0 warnings

---

### Feature: Validate Skill

#### Scenario: Skill converts .allium and validates

**Traces to**: User Story 10, Acceptance Scenario 1
**Category**: Happy Path

- **Given** a `.allium` file (human-readable Allium syntax)
- **When** the validate skill is invoked on it
- **Then** a `.allium.json` file is generated alongside
- **And** `allium-check` is run on the generated JSON
- **And** results are presented to the user

---

#### Scenario: Skill presents errors with suggested fixes

**Traces to**: User Story 10, Acceptance Scenario 2
**Category**: Alternate Path

- **Given** `allium-check` returns errors including "Entity 'Candidte' referenced but not declared"
- **When** the skill interprets the results
- **Then** the error is presented with a suggestion: "Did you mean 'Candidate'?"

---

#### Scenario: Guidance check flags vague naming

**Traces to**: User Story 10, Acceptance Scenarios 3-4
**Category**: Alternate Path

- **Given** `allium-check` returns clean (no errors, no warnings)
- **And** the spec contains rules named "DoThing" and "HandleStuff"
- **When** guidance checks run
- **Then** the skill flags each vague name: "Rule name 'DoThing' is not descriptive — consider naming it after the business event it handles"

---

#### Scenario: Guidance check suggests missing patterns

**Traces to**: User Story 10, Acceptance Scenario 5
**Category**: Alternate Path

- **Given** `allium-check` returns clean
- **And** the spec has entities with Timestamp fields but no temporal rules
- **When** guidance checks run
- **Then** the skill suggests "Entity 'Invitation' has 'expires_at' but no temporal rule — consider adding an expiry rule"

---

#### Scenario: Skill handles missing CLI binary

**Traces to**: User Story 10, Acceptance Scenario 6
**Category**: Error Path

- **Given** `allium-check` is not found in PATH
- **When** the validate skill is invoked
- **Then** the skill reports "allium-check binary not found. Build it with: go build -o bin/allium-check ./cmd/allium-check"

---

### Feature: Skill Integration

#### Scenario: Distill skill auto-validates output

**Traces to**: User Story 11, Acceptance Scenarios 1-2
**Category**: Happy Path

- **Given** the distill skill has completed Step 7 (spec output)
- **When** Step 8 runs
- **Then** a `.allium.json` is generated from the `.allium` file
- **And** `allium-check` is invoked on it
- **And** if errors are found, the skill fixes them before presenting to the user

---

#### Scenario: Elicit skill uses validation errors as prompts

**Traces to**: User Story 11, Acceptance Scenarios 3-4
**Category**: Alternate Path

- **Given** the elicit skill is in Phase 4 (Refinement)
- **And** validation of the current spec finds "Entity 'Order' has unreachable status value 'archived'"
- **When** the error is processed
- **Then** the skill asks the user: "Entity Order has a status value 'archived' that no rule ever reaches. Is that intentional?"

---

### Feature: Reference Example

#### Scenario: Reference example passes all checks

**Traces to**: User Story 12, Acceptance Scenarios 1-2
**Category**: Happy Path

- **Given** `schemas/v1/examples/password-auth.allium.json` exists
- **And** it represents the password-auth pattern with entities, rules, surfaces, actors, config, and diverse trigger types
- **When** `allium-check password-auth.allium.json` is run
- **Then** exit code is 0
- **And** output shows 0 errors and 0 warnings

---

#### Scenario Outline: Broken variants produce correct errors

**Traces to**: User Story 12, Acceptance Scenario 3
**Category**: Error Path

- **Given** a modified `password-auth.allium.json` with `<modification>`
- **When** `allium-check broken-variant.allium.json` is run
- **Then** error `<expected_rule>` is reported

**Examples**:

| modification | expected_rule |
|-------------|---------------|
| entity_ref to undeclared "FooBar" | RULE-01 |
| Duplicate config parameter name | RULE-26 |
| Unreachable status value | RULE-07 |
| Circular derived values | RULE-10 |
| Unguarded variant field access | RULE-18 |

---

### Feature: Validation Documentation

#### Scenario: Complete rule documentation exists

**Traces to**: User Story 13, Acceptance Scenarios 1-3
**Category**: Happy Path

- **Given** the validation documentation files exist
- **When** `VALIDATION-RULES.md` is read
- **Then** every rule RULE-01 through RULE-35 and warning WARN-01 through WARN-19 has an entry
- **And** each entry includes ID, description, severity, and implementation mapping
- **And** each rule group document contains explanations, example violations, and example fixes

---

## Test-Driven Development Plan

### Test Hierarchy

| Level       | Scope                                            | Purpose                                                |
|-------------|--------------------------------------------------|--------------------------------------------------------|
| Unit        | Individual schema files, AST loading, each check function, each warning, output formatters | Validates logic in isolation |
| Integration | Full checker pipeline (schema + semantic passes), CLI binary, multi-file validation | Validates components work together |
| E2E         | Reference example, validate skill workflow, broken variant detection | Validates complete feature from user perspective |

### Test Implementation Order

Write these tests BEFORE implementing the feature code. Order: unit first, then integration, then E2E. Within each level, order by dependency.

| Order | Test Name | Level | Traces to BDD Scenario | Description |
|-------|-----------|-------|------------------------|-------------|
| 1 | TestLoadValidSpec | Unit | Complete valid document passes | JSON deserialization of well-formed `.allium.json` into Go structs |
| 2 | TestLoadInvalidJSON | Unit | CLI handles input errors | Rejects malformed JSON with parse error |
| 3 | TestSchemaValidatesCompleteDocument | Unit | Complete valid document passes | Embedded schema accepts valid document |
| 4 | TestSchemaRejectsMissingFields | Unit | Schema rejects missing required fields | Missing version, file, etc. produce schema errors |
| 5 | TestSchemaEnforcesRule02 | Unit | Schema enforces structural rules (Rule 2) | Fields require name + type |
| 6 | TestSchemaEnforcesRule04 | Unit | Schema enforces structural rules (Rule 4) | Rules require trigger + non-empty ensures |
| 7 | TestSchemaEnforcesRule05 | Unit | Schema enforces structural rules (Rule 5) | Trigger kind must be one of 7 |
| 8 | TestSchemaEnforcesRule15 | Unit | Schema enforces structural rules (Rule 15) | Variant names must be PascalCase |
| 9 | TestSchemaEnforcesRule21 | Unit | Schema enforces structural rules (Rule 21) | Variant requires name + base_entity |
| 10 | TestSchemaEnforcesRule25 | Unit | Schema enforces structural rules (Rule 25) | Config requires name, type, default |
| 11 | TestSchemaEnforcesNamingPatterns | Unit | Schema enforces naming patterns | PascalCase/snake_case enforcement |
| 12 | TestSchemaPerConstructFile | Unit | Each schema definition file validates | 14 schema files accept valid constructs |
| 13 | TestBuildSymbolTable | Unit | All references resolve successfully | Symbol table collects all names correctly |
| 14 | TestReferenceRule01 | Unit | Undeclared references detected (Rule 1) | entity_ref to undeclared entity |
| 15 | TestReferenceRule03 | Unit | Undeclared references detected (Rule 3) | Relationship target unresolved |
| 16 | TestReferenceRule22 | Unit | Undeclared references detected (Rule 22) | Given type unresolved |
| 17 | TestReferenceRule27 | Unit | Undeclared references detected (Rule 27) | Config reference unresolved |
| 18 | TestReferenceRule28 | Unit | Undeclared references detected (Rule 28) | Surface facing unresolved |
| 19 | TestReferenceRule30 | Unit | Undeclared references detected (Rule 30) | Provides trigger unresolved |
| 20 | TestReferenceRule31 | Unit | Undeclared references detected (Rule 31) | Related surface unresolved |
| 21 | TestReferenceRule35 | Unit | Undeclared references detected (Rule 35) | Use declaration import unresolved |
| 22 | TestUniquenessRule06Incompatible | Unit | Incompatible trigger parameter signatures | Same trigger, different params |
| 23 | TestUniquenessRule06Compatible | Unit | Compatible trigger signatures pass | Same trigger, matching params |
| 24 | TestUniquenessRule23 | Unit | Duplicate names detected (Rule 23) | Duplicate given binding |
| 25 | TestUniquenessRule26 | Unit | Duplicate names detected (Rule 26) | Duplicate config parameter |
| 26 | TestStateMachineRule07 | Unit | Unreachable status value | BFS from creation points misses value |
| 27 | TestStateMachineRule08 | Unit | Dead-end non-terminal state | Non-terminal with no outgoing edge |
| 28 | TestStateMachineRule09 | Unit | Undeclared status value | Ensures assigns undeclared value |
| 29 | TestStateMachineValid | Unit | Valid state machine passes | All states reachable, no dead ends |
| 30 | TestExpressionRule10 | Unit | Circular derived dependency | Tarjan's SCC finds cycle |
| 31 | TestExpressionRule11 | Unit | Out-of-scope field access | Unknown binding root |
| 32 | TestExpressionRule12Types | Unit | Type mismatches in expressions | Integer vs String comparison |
| 33 | TestExpressionRule12Arithmetic | Unit | Type mismatches (arithmetic) | Boolean + Integer |
| 34 | TestExpressionRule12ValidTemporal | Unit | Type mismatches (valid temporal) | Timestamp - Duration accepted |
| 35 | TestExpressionRule13 | Unit | Missing lambda parameter | any/all without lambda_param |
| 36 | TestExpressionRule14Inline | Unit | Inline enum cross-comparison | Two inline enums compared |
| 37 | TestExpressionRule14Named | Unit | Named enum comparison accepted | Same named enum accepted |
| 38 | TestSumTypeRule16 | Unit | Missing variant declaration | Discriminator variant undeclared |
| 39 | TestSumTypeRule17 | Unit | Variant not in discriminator | Variant not listed |
| 40 | TestSumTypeRule18Unguarded | Unit | Unguarded variant field access | No type guard active |
| 41 | TestSumTypeRule18Guarded | Unit | Guarded variant field access | Type guard active, no error |
| 42 | TestSumTypeRule19 | Unit | Entity creation must use variant | Base name used instead of variant |
| 43 | TestSurfaceRule29 | Unit | Unreachable exposes path | Path not reachable from bindings |
| 44 | TestSurfaceRule32 | Unit | Unused binding | Facing binding never referenced |
| 45 | TestSurfaceRule33 | Unit | Invalid when condition reference | When references unknown field |
| 46 | TestSurfaceRule34 | Unit | Non-collection iteration | For-loop on String field |
| 47 | TestWarn01ExternalNoSpec | Unit | Structural warnings (WARN-01) | External entity without use decl |
| 48 | TestWarn02OpenQuestions | Unit | Structural warnings (WARN-02) | Non-empty open_questions |
| 49 | TestWarn03DeferredNoHint | Unit | Structural warnings (WARN-03) | Deferred spec no location_hint |
| 50 | TestWarn04UnusedEntity | Unit | Structural warnings (WARN-04) | Entity never referenced |
| 51 | TestWarn05NeverFires | Unit | Rule quality warnings (WARN-05) | Contradictory requires |
| 52 | TestWarn06TemporalNoGuard | Unit | Rule quality warnings (WARN-06) | Temporal without re-firing guard |
| 53 | TestWarn07UnusedExposed | Unit | Surface warnings (WARN-07) | Exposed field used by no rule |
| 54 | TestWarn08ImpossibleProvides | Unit | Surface warnings (WARN-08) | Always-false when condition |
| 55 | TestWarn09UnusedActor | Unit | Surface warnings (WARN-09) | Actor not in any facing |
| 56 | TestWarn10SiblingCreation | Unit | Rule quality warnings (WARN-10) | Child creation without guard |
| 57 | TestWarn11WeakProvides | Unit | Surface warnings (WARN-11) | Provides weaker than requires |
| 58 | TestWarn12OverlappingRequires | Unit | Rule quality warnings (WARN-12) | Overlapping preconditions |
| 59 | TestWarn13DerivedScope | Unit | Expression warnings (WARN-13) | Out-of-entity reference |
| 60 | TestWarn14TrivialActor | Unit | Expression warnings (WARN-14) | Always-true identified_by |
| 61 | TestWarn15EmptyConditionalPath | Unit | Rule quality warnings (WARN-15) | All conditional, one empty |
| 62 | TestWarn16OptionalTemporal | Unit | Expression warnings (WARN-16) | Temporal on optional field |
| 63 | TestWarn17RawWithActors | Unit | Surface warnings (WARN-17) | Raw entity in facing |
| 64 | TestWarn18TransitionsOnCreation | Unit | Rule quality warnings (WARN-18) | transitions_to on creation value |
| 65 | TestWarn19DuplicateInlineEnums | Unit | Expression warnings (WARN-19) | Identical inline enum sets |
| 66 | TestNoWarningsCleanSpec | Unit | Clean spec produces no warnings | No warning-triggering patterns |
| 67 | TestFormatJSON | Unit | CLI outputs JSON format | JSON output matches report schema |
| 68 | TestFormatText | Unit | CLI reports clean validation | Text output format |
| 69 | TestCheckerFullPipeline | Integration | All references resolve successfully | Schema + all 7 semantic passes on valid spec |
| 70 | TestCLIExitCode0 | Integration | CLI reports clean validation | allium-check returns 0 for clean file |
| 71 | TestCLIExitCode1 | Integration | CLI reports validation errors | allium-check returns 1 for errors |
| 72 | TestCLIExitCode2 | Integration | CLI handles input errors | allium-check returns 2 for bad input |
| 73 | TestCLIFlags | Integration | CLI flag behavior | --quiet, --strict, --schema-only, --rules |
| 74 | TestCLIMultiFile | Integration | CLI validates multiple files | Per-file results |
| 75 | TestReferenceExampleClean | E2E | Reference example passes all checks | password-auth.allium.json passes |
| 76 | TestReferenceExampleBrokenVariants | E2E | Broken variants produce correct errors | Each broken variant reports correct rule |
| 77 | TestFullValidationPipeline | E2E | Checker full pipeline | Schema + semantic + warnings on reference example |

### Test Datasets

#### Dataset: Document Structure Inputs

| # | Input | Boundary Type | Expected Output | Traces to | Notes |
|---|-------|---------------|-----------------|-----------|-------|
| 1 | `{}` | Empty object | Schema error: missing required fields | BDD: Schema rejects missing fields | All top-level fields absent |
| 2 | `[]` | Wrong root type | Schema error: root must be object | BDD: Schema rejects missing fields | Array instead of object |
| 3 | `""` | Wrong root type | Schema error: root must be object | BDD: Schema rejects missing fields | String instead of object |
| 4 | `{"version": 1, "file": "test.allium", ...}` with all required fields | Valid minimum | Schema passes | BDD: Complete valid document | Minimal valid document |
| 5 | `{"version": "1", ...}` | Wrong type for version | Schema error: version must be integer | BDD: Schema rejects missing fields | String instead of integer |
| 6 | `{"version": 0, ...}` | Boundary: version = 0 | Schema error or passes depending on min constraint | BDD: Schema rejects missing fields | Zero version |
| 7 | Document with 20 entities, 30 rules, 5 surfaces | Representative large | Schema passes | BDD: Complete valid document | Stress test (still small) |
| 8 | `null` | Null input | Parse error (exit 2) | BDD: CLI handles input errors | Not valid JSON object |

#### Dataset: Naming Pattern Inputs

| # | Input | Boundary Type | Expected Output | Traces to | Notes |
|---|-------|---------------|-----------------|-----------|-------|
| 1 | Entity name: `"User"` | Valid PascalCase | Passes | BDD: Schema enforces naming | Single word |
| 2 | Entity name: `"InterviewSlot"` | Valid PascalCase | Passes | BDD: Schema enforces naming | Multi-word |
| 3 | Entity name: `"user"` | Invalid: lowercase | Schema error | BDD: Schema enforces naming | Starts lowercase |
| 4 | Entity name: `"interview_slot"` | Invalid: snake_case | Schema error | BDD: Schema enforces naming | Underscores |
| 5 | Entity name: `"A"` | Valid: single char PascalCase | Passes | BDD: Schema enforces naming | Minimum length |
| 6 | Entity name: `""` | Empty | Schema error | BDD: Schema enforces naming | Zero length |
| 7 | Field name: `"email_address"` | Valid snake_case | Passes | BDD: Schema enforces naming | Standard |
| 8 | Field name: `"emailAddress"` | Invalid: camelCase | Schema error | BDD: Schema enforces naming | No camelCase |
| 9 | Field name: `"EmailAddress"` | Invalid: PascalCase | Schema error | BDD: Schema enforces naming | Wrong case |
| 10 | Field name: `"a"` | Valid: single char | Passes | BDD: Schema enforces naming | Minimum |
| 11 | Enum literal: `"pending"` | Valid snake_case | Passes | BDD: Schema enforces naming | Standard |
| 12 | Enum literal: `"Pending"` | Invalid: PascalCase | Schema error | BDD: Schema enforces naming | Wrong case |

#### Dataset: CLI Flag Combinations

| # | Input Flags | File State | Expected Exit Code | Expected Behavior | Traces to | Notes |
|---|-------------|-----------|-------------------|-------------------|-----------|-------|
| 1 | (none) | Clean | 0 | Text output, 0 errors, 0 warnings | BDD: CLI reports clean | Default behavior |
| 2 | `--format json` | Clean | 0 | JSON output, valid schema | BDD: CLI outputs JSON | JSON format |
| 3 | `--format text` | 1 error | 1 | Text with error details | BDD: CLI reports errors | Explicit text |
| 4 | `--quiet` | 1 warning | 0 | No warning shown | BDD: CLI flag behavior (quiet) | Suppresses warnings |
| 5 | `--strict` | 1 warning | 1 | Warning treated as error | BDD: CLI flag behavior (strict) | Elevates warnings |
| 6 | `--quiet --strict` | 1 warning | 1 | Warning not shown but exit 1 | BDD: Edge case | Suppressed but counted |
| 7 | `--schema-only` | Schema + semantic errors | 1 | Only schema error | BDD: CLI flag behavior (schema-only) | Skips semantic |
| 8 | `--rules 7-9` | Errors on rules 7, 10 | 1 | Only rule 7 reported | BDD: CLI flag behavior (rules) | Selective rules |
| 9 | `--version` | Any | 0 | Version printed | BDD: CLI flag behavior (version) | No validation |
| 10 | `--format invalid` | Any | 2 | Unknown format error | BDD: CLI handles input errors | Bad flag value |

#### Dataset: State Machine Inputs

| # | Entity Status Values | Creation Value | Transitions | Expected | Traces to | Notes |
|---|---------------------|---------------|------------|----------|-----------|-------|
| 1 | `pending, active, done` | `pending` | `pending->active, active->done` | Clean | BDD: Valid state machine | All reachable |
| 2 | `pending, active, done, archived` | `pending` | `pending->active, active->done` | RULE-07: archived unreachable | BDD: Unreachable status | Dead value |
| 3 | `open, blocked, done` | `open` | `open->blocked` | RULE-08: blocked dead-end | BDD: Dead-end state | No exit from blocked |
| 4 | `pending, active` | `pending` | `pending->completed` | RULE-09: completed undeclared | BDD: Undeclared value | Not in enum |
| 5 | `a, b, c, d, e` | `a` | `a->b, b->c, c->d, d->e` | Clean | BDD: Valid state machine | Linear chain |
| 6 | `a, b` | `a` | `a->b, b->a` | Clean | BDD: Valid state machine | Cycle is valid |
| 7 | (no enum field) | N/A | N/A | Skipped | BDD: Valid state machine | Not applicable |

#### Dataset: Expression Type Compatibility

| # | Left Type | Operation | Right Type | Expected | Traces to | Notes |
|---|----------|-----------|-----------|----------|-----------|-------|
| 1 | Integer | `=` | Integer | Valid | BDD: Named enum accepted | Same type |
| 2 | Integer | `=` | String | RULE-12 | BDD: Type mismatches | Incompatible |
| 3 | Boolean | `+` | Integer | RULE-12 | BDD: Type mismatches | Non-numeric arithmetic |
| 4 | Timestamp | `-` | Duration | Valid | BDD: Type mismatches (valid) | Temporal arithmetic |
| 5 | Timestamp | `+` | Duration | Valid | BDD: Type mismatches (valid) | Temporal arithmetic |
| 6 | Duration | `+` | Duration | Valid | BDD: Type mismatches (valid) | Duration addition |
| 7 | String | `<` | String | Valid | BDD: Named enum accepted | String comparison |
| 8 | Integer | `/` | Integer | Valid | BDD: Named enum accepted | Numeric arithmetic |
| 9 | Decimal | `*` | Integer | Valid | BDD: Named enum accepted | Mixed numeric |
| 10 | inline_enum | `=` | inline_enum (different) | RULE-14 | BDD: Inline enum cross | Cross-comparison |
| 11 | named_enum(Priority) | `=` | named_enum(Priority) | Valid | BDD: Named enum accepted | Same named enum |
| 12 | named_enum(Priority) | `=` | named_enum(Status) | RULE-14 | BDD: Inline enum cross | Different named enums |

### Regression Test Requirements

> No regression impact — new capability. The allium-check CLI, JSON Schema definitions, semantic validation, and validate skill are entirely new. There is no existing code in the repository to protect.
>
> Integration seams to protect:
> - Existing `.allium` files in the patterns library — the validator must not require changes to the human-readable format
> - Existing skill definitions (distill, elicit) — modifications (US-11) add a new step but must not break existing steps
> - Language reference documentation — the validator must agree with the language-reference.md specification; any discrepancy is a validator bug

---

## Functional Requirements

### Schema Validation

- **FR-001**: System MUST embed all 14 JSON Schema definition files via `go:embed` at build time.
- **FR-002**: System MUST validate `.allium.json` documents against the root schema `allium-spec.json` which composes all definition files via `$ref`.
- **FR-003**: System MUST enforce Rule 2 (fields require name + type) via the Entity Field schema.
- **FR-004**: System MUST enforce Rule 4 (rules require trigger + non-empty ensures) via the Rule schema with `minItems: 1`.
- **FR-005**: System MUST enforce Rule 5 (trigger types valid) via the Trigger schema with `oneOf` discriminated by `kind`.
- **FR-006**: System MUST enforce Rule 15 (discriminator variants PascalCase) via naming pattern regex.
- **FR-007**: System MUST enforce Rule 20 (discriminator names follow naming pattern) via `snake_case_name` pattern.
- **FR-008**: System MUST enforce Rule 21 (variant declaration requires name + base_entity) via Variant schema `required`.
- **FR-009**: System MUST enforce Rule 25 (config requires name, type, default_value) via ConfigParameter schema `required`.
- **FR-010**: System MUST enforce PascalCase pattern (`^[A-Z][a-zA-Z0-9]*$`) for entity names, variant names, rule names, trigger names, actor names, and surface names.
- **FR-011**: System MUST enforce snake_case pattern (`^[a-z][a-z0-9_]*$`) for field names, config parameters, derived values, enum literals, and relationship names.

### CLI Interface

- **FR-012**: System MUST accept one or more `.allium.json` file paths as positional arguments.
- **FR-013**: System MUST return exit code 0 when no errors are found, 1 when validation errors exist, and 2 for input/parse errors.
- **FR-014**: System MUST support `--format text` (default) and `--format json` output formats.
- **FR-015**: System MUST support `--quiet` flag to suppress warnings from output.
- **FR-016**: System MUST support `--strict` flag to treat warnings as errors (exit code 1 if any warnings).
- **FR-017**: System MUST support `--schema-only` flag to skip semantic validation.
- **FR-018**: System MUST support `--rules` flag accepting comma-separated rule IDs and ranges.
- **FR-019**: System MUST support `--version` flag to print version and exit.
- **FR-020**: System MUST validate multiple files independently and report results per file.
- **FR-021**: JSON output MUST include keys: `file`, `schema_valid`, `errors`, `warnings`, `summary`.
- **FR-022**: Each error/warning MUST include `rule` (ID), `severity`, `message`, and `location`.

### Reference Resolution (Semantic Pass 1)

- **FR-023**: System MUST verify that all `entity_ref` type references resolve to a declared entity, external entity, or imported type (Rule 1).
- **FR-024**: System MUST verify that all relationship `target_entity` values resolve to a declared entity (Rule 3).
- **FR-025**: System MUST verify that all `given` binding types resolve to declared entities or value types (Rule 22).
- **FR-026**: System MUST verify that all config parameter references in expressions resolve to declared config parameters (Rule 27).
- **FR-027**: System MUST verify that all surface `facing` types resolve to declared entities or actors (Rule 28).
- **FR-028**: System MUST verify that all surface `provides` trigger names resolve to declared rule triggers (Rule 30).
- **FR-029**: System MUST verify that all surface `related` surface names resolve to declared surfaces (Rule 31).
- **FR-030**: System MUST verify that all `use_declaration` imported types resolve (Rule 35).

### Uniqueness (Semantic Pass 2)

- **FR-031**: System MUST verify that rules sharing a trigger name have the same parameter count and positional types (Rule 6).
- **FR-032**: System MUST verify that no duplicate names exist in `given` bindings (Rule 23).
- **FR-033**: System MUST verify that no duplicate names exist in `config` parameters (Rule 26).

### State Machine Analysis (Semantic Pass 3)

- **FR-034**: System MUST build a state machine graph for each entity with an enum-typed status field by collecting creation points and transition rules.
- **FR-035**: System MUST report unreachable status values (values not reachable via BFS from creation values) as RULE-07 errors (Rule 7).
- **FR-036**: System MUST report non-terminal status values with no outgoing transition as RULE-08 errors (Rule 8).
- **FR-037**: System MUST report ensures clauses that assign undeclared enum values as RULE-09 errors (Rule 9).

### Expression Analysis (Semantic Pass 4)

- **FR-038**: System MUST detect cycles in derived value dependencies using Tarjan's SCC algorithm and report as RULE-10 errors (Rule 10).
- **FR-039**: System MUST verify that every `field_access` path root is in scope (trigger bindings, for-clause, let, given, defaults) and report violations as RULE-11 errors (Rule 11).
- **FR-040**: System MUST verify type compatibility in comparisons and arithmetic, reporting mismatches as RULE-12 errors (Rule 12).
- **FR-041**: System MUST verify that `any`/`all` expressions have explicit `lambda_param` fields, reporting omissions as RULE-13 errors (Rule 13).
- **FR-042**: System MUST reject comparisons between inline enum fields and restrict named enum comparisons to same-type enums, reporting violations as RULE-14 errors (Rule 14).

### Sum Type Analysis (Semantic Pass 5)

- **FR-043**: System MUST verify that every discriminator variant name has a corresponding `variant` declaration, reporting missing declarations as RULE-16 errors (Rule 16).
- **FR-044**: System MUST verify that every variant is listed in its base entity's discriminator, reporting unlisted variants as RULE-17 errors (Rule 17).
- **FR-045**: System MUST verify that variant-specific field access occurs only within type guards (requires clauses or if branches checking the discriminator), reporting unguarded access as RULE-18 errors (Rule 18).
- **FR-046**: System MUST verify that entity creation uses variant names when a discriminator exists, reporting base entity name usage as RULE-19 errors (Rule 19).

### Surface Analysis (Semantic Pass 6)

- **FR-047**: System MUST verify that all `exposes` field paths are reachable from facing, context, or let bindings, reporting unreachable paths as RULE-29 errors (Rule 29).
- **FR-048**: System MUST verify that facing and context bindings are referenced somewhere in the surface body, reporting unused bindings as RULE-32 errors (Rule 32).
- **FR-049**: System MUST verify that `when` conditions reference fields reachable from party or context bindings, reporting invalid references as RULE-33 errors (Rule 33).
- **FR-050**: System MUST verify that `for` iterations target collection-typed fields/bindings, reporting non-collection targets as RULE-34 errors (Rule 34).

### Warning Checks (Semantic Pass 7)

- **FR-051**: System MUST detect and report each of the 19 warning conditions (WARN-01 through WARN-19) as warning-severity findings.
- **FR-052**: System MUST categorize warnings separately from errors in all output formats.
- **FR-053**: Warnings MUST NOT cause a non-zero exit code unless `--strict` flag is used.

### Validate Skill

- **FR-054**: The validate skill MUST convert `.allium` files to `.allium.json` before running `allium-check`.
- **FR-055**: The validate skill MUST invoke `allium-check --format json` and parse the structured output.
- **FR-056**: The validate skill MUST present each error with a human-readable explanation and a suggested fix.
- **FR-057**: The validate skill MUST run LLM guidance checks for naming quality when `allium-check` returns clean.
- **FR-058**: The validate skill MUST run LLM guidance checks for spec completeness (missing common patterns).
- **FR-059**: The validate skill MUST report a clear error message when `allium-check` binary is not found.

### Skill Integration

- **FR-060**: The distill skill MUST add a Step 8 that generates `.allium.json` and invokes validation, fixing errors before presenting output.
- **FR-061**: The elicit skill MUST generate `.allium.json` after session output and use validation errors as conversation prompts during Phase 4.
- **FR-062**: The root `SKILL.md` MUST include a routing table entry for the `validate` skill.

### Reference Example

- **FR-063**: A reference example `schemas/v1/examples/password-auth.allium.json` MUST exist, exercising all major construct types.
- **FR-064**: The reference example MUST pass `allium-check` with 0 errors and 0 warnings.

### Documentation

- **FR-065**: `VALIDATION-RULES.md` MUST map every rule (1-35) and warning (1-19) to its implementation with ID, description, and severity.
- **FR-066**: Each of 7 rule group documents MUST explain each rule with an example violation and example fix.
- **FR-067**: `warnings.md` MUST explain each warning with an example trigger and suggested resolution.

---

## Success Criteria

- **SC-001**: `allium-check password-auth.allium.json` exits with code 0, producing 0 errors and 0 warnings.
- **SC-002**: `go test ./...` passes with 100% of tests passing (0 failures) for the allium module.
- **SC-003**: Each of the 35 semantic validation rules has at least one dedicated test case that confirms the rule detects its target violation.
- **SC-004**: Each of the 19 warning checks has at least one dedicated test case that confirms the warning fires for its target pattern.
- **SC-005**: For every intentionally broken variant of the reference example, `allium-check` reports the correct rule ID for the specific violation introduced.
- **SC-006**: JSON output from `allium-check --format json` is valid JSON parseable by `jq` in all cases (clean, errors, warnings, input errors).
- **SC-007**: `allium-check --schema-only` runs in under 100ms for the reference example (schema validation is fast).
- **SC-008**: `allium-check` (full validation) completes in under 2 seconds for any spec with fewer than 20 entities.
- **SC-009**: The `allium-check` binary builds with `go build` using only the declared dependency (`santhosh-tekuri/jsonschema/v6`) plus standard library.
- **SC-010**: The validate skill successfully converts at least one `.allium` file to `.allium.json`, runs `allium-check`, and presents results without manual intervention.
- **SC-011**: The distill skill's Step 8 and elicit skill's validation integration produce a `.allium.json` file alongside every `.allium` file they generate.
- **SC-012**: Every rule ID (RULE-01 through RULE-35) and warning ID (WARN-01 through WARN-19) appears in `VALIDATION-RULES.md` with a description.

---

## Traceability Matrix

| Requirement | User Story | BDD Scenario(s) | Test Name(s) |
|-------------|-----------|------------------|---------------|
| FR-001 | US-1 | Complete valid document passes | TestSchemaValidatesCompleteDocument |
| FR-002 | US-1 | Complete valid document passes | TestSchemaValidatesCompleteDocument |
| FR-003 | US-1 | Schema enforces structural rules (Rule 2) | TestSchemaEnforcesRule02 |
| FR-004 | US-1 | Schema enforces structural rules (Rule 4) | TestSchemaEnforcesRule04 |
| FR-005 | US-1 | Schema enforces structural rules (Rule 5) | TestSchemaEnforcesRule05 |
| FR-006 | US-1 | Schema enforces structural rules (Rule 15) | TestSchemaEnforcesRule15 |
| FR-007 | US-1 | Schema enforces naming patterns | TestSchemaEnforcesNamingPatterns |
| FR-008 | US-1 | Schema enforces structural rules (Rule 21) | TestSchemaEnforcesRule21 |
| FR-009 | US-1 | Schema enforces structural rules (Rule 25) | TestSchemaEnforcesRule25 |
| FR-010 | US-1 | Schema enforces naming patterns | TestSchemaEnforcesNamingPatterns |
| FR-011 | US-1 | Schema enforces naming patterns | TestSchemaEnforcesNamingPatterns |
| FR-012 | US-2 | CLI reports clean validation, CLI validates multiple files | TestCLIExitCode0, TestCLIMultiFile |
| FR-013 | US-2 | CLI reports clean/errors/input errors | TestCLIExitCode0, TestCLIExitCode1, TestCLIExitCode2 |
| FR-014 | US-2 | CLI outputs JSON format, CLI reports clean | TestFormatJSON, TestFormatText |
| FR-015 | US-2 | CLI flag behavior (quiet) | TestCLIFlags |
| FR-016 | US-2 | CLI flag behavior (strict) | TestCLIFlags |
| FR-017 | US-2 | CLI flag behavior (schema-only) | TestCLIFlags |
| FR-018 | US-2 | CLI flag behavior (rules) | TestCLIFlags |
| FR-019 | US-2 | CLI flag behavior (version) | TestCLIFlags |
| FR-020 | US-2 | CLI validates multiple files | TestCLIMultiFile |
| FR-021 | US-2 | CLI outputs JSON format | TestFormatJSON |
| FR-022 | US-2 | CLI reports validation errors | TestCLIExitCode1, TestFormatJSON |
| FR-023 | US-3 | Undeclared references (Rule 1) | TestReferenceRule01 |
| FR-024 | US-3 | Undeclared references (Rule 3) | TestReferenceRule03 |
| FR-025 | US-3 | Undeclared references (Rule 22) | TestReferenceRule22 |
| FR-026 | US-3 | Undeclared references (Rule 27) | TestReferenceRule27 |
| FR-027 | US-3 | Undeclared references (Rule 28) | TestReferenceRule28 |
| FR-028 | US-3 | Undeclared references (Rule 30) | TestReferenceRule30 |
| FR-029 | US-3 | Undeclared references (Rule 31) | TestReferenceRule31 |
| FR-030 | US-3 | Undeclared references (Rule 35) | TestReferenceRule35 |
| FR-031 | US-4 | Incompatible trigger signatures | TestUniquenessRule06Incompatible |
| FR-032 | US-4 | Duplicate names (Rule 23) | TestUniquenessRule23 |
| FR-033 | US-4 | Duplicate names (Rule 26) | TestUniquenessRule26 |
| FR-034 | US-5 | Valid state machine, Unreachable status | TestStateMachineRule07, TestStateMachineValid |
| FR-035 | US-5 | Unreachable status value | TestStateMachineRule07 |
| FR-036 | US-5 | Dead-end non-terminal state | TestStateMachineRule08 |
| FR-037 | US-5 | Undeclared status value | TestStateMachineRule09 |
| FR-038 | US-6 | Circular derived dependency | TestExpressionRule10 |
| FR-039 | US-6 | Out-of-scope field access | TestExpressionRule11 |
| FR-040 | US-6 | Type mismatches in expressions | TestExpressionRule12Types, TestExpressionRule12Arithmetic |
| FR-041 | US-6 | Missing lambda parameter | TestExpressionRule13 |
| FR-042 | US-6 | Inline enum cross-comparison, Named enum accepted | TestExpressionRule14Inline, TestExpressionRule14Named |
| FR-043 | US-7 | Missing variant declaration | TestSumTypeRule16 |
| FR-044 | US-7 | Variant not in discriminator | TestSumTypeRule17 |
| FR-045 | US-7 | Unguarded/guarded variant field access | TestSumTypeRule18Unguarded, TestSumTypeRule18Guarded |
| FR-046 | US-7 | Entity creation must use variant | TestSumTypeRule19 |
| FR-047 | US-8 | Unreachable exposes path | TestSurfaceRule29 |
| FR-048 | US-8 | Unused binding | TestSurfaceRule32 |
| FR-049 | US-8 | Invalid when condition reference | TestSurfaceRule33 |
| FR-050 | US-8 | Non-collection iteration | TestSurfaceRule34 |
| FR-051 | US-9 | All warning scenario outlines (WARN-01 to WARN-19) | TestWarn01 through TestWarn19 |
| FR-052 | US-9 | Clean spec produces no warnings | TestNoWarningsCleanSpec, TestFormatJSON |
| FR-053 | US-9, US-2 | CLI flag behavior (strict/quiet) | TestCLIFlags |
| FR-054 | US-10 | Skill converts .allium and validates | (Skill test — manual/LLM verification) |
| FR-055 | US-10 | Skill converts .allium and validates | (Skill test — manual/LLM verification) |
| FR-056 | US-10 | Skill presents errors with suggested fixes | (Skill test — manual/LLM verification) |
| FR-057 | US-10 | Guidance check flags vague naming | (Skill test — manual/LLM verification) |
| FR-058 | US-10 | Guidance check suggests missing patterns | (Skill test — manual/LLM verification) |
| FR-059 | US-10 | Skill handles missing CLI binary | (Skill test — manual/LLM verification) |
| FR-060 | US-11 | Distill skill auto-validates output | (Skill integration test — manual) |
| FR-061 | US-11 | Elicit skill uses validation errors as prompts | (Skill integration test — manual) |
| FR-062 | US-11 | (Routing table entry) | (Manual verification) |
| FR-063 | US-12 | Reference example passes all checks | TestReferenceExampleClean |
| FR-064 | US-12 | Reference example passes all checks | TestReferenceExampleClean |
| FR-065 | US-13 | Complete rule documentation exists | (Manual review) |
| FR-066 | US-13 | Complete rule documentation exists | (Manual review) |
| FR-067 | US-13 | Complete rule documentation exists | (Manual review) |

**Completeness check**: Every FR-xxx row has at least one BDD scenario and one test (or explicit manual verification for LLM skill tests). Every BDD scenario appears in at least one row.

---

## Assumptions

- The Allium language specification (`references/language-reference.md`) is the authoritative source of truth for validation rules. Any discrepancy between the spec and the validator is a validator bug.
- Go 1.21+ is available for building the CLI binary.
- The `github.com/santhosh-tekuri/jsonschema/v6` library supports JSON Schema draft 2020-12 and handles `$ref` composition correctly.
- `.allium.json` files are always valid JSON (the CLI handles invalid JSON as exit code 2, not as a validation error).
- `.allium` to `.allium.json` conversion is performed by LLM skills, not by the CLI. The CLI only reads `.allium.json`.
- Spec sizes are small (<20 entities, <50 rules) and performance is not a constraint.
- SARIF output format is deferred to a future iteration (P3+).
- The password-auth pattern from `references/patterns.md` is sufficiently representative to serve as the reference example.
- Cross-file validation (validating references between multiple `.allium.json` files from different specs) is out of scope.
- The `golang-expert-coder` agent is used for all Go implementation work per project conventions.

## Clarifications

### 2026-02-15

- Q: Should the spec cover the entire 7-phase plan or a subset? -> A: Entire plan (all 7 phases) in one spec.
- Q: Who are the primary actors? -> A: All three — LLM skills, human spec authors, and CI pipelines.
- Q: What are the performance constraints? -> A: Small specs only (<20 entities). Performance is not a concern.
- Q: Is there an existing reference `.allium.json` example? -> A: No — it must be created as part of this work.
- Q: Is SARIF output required for initial release? -> A: Deferred to P3+. JSON and text are sufficient initially.
- Q: Should the validate skill's guidance checks be detailed? -> A: Yes — enumerate each guidance check with acceptance criteria.
- Q: Should warnings have full BDD rigor? -> A: Yes — each warning gets individual BDD scenarios and tests.
- Q: Should schema acceptance be per-file or per-deliverable? -> A: Per-file — each of the 14 schema definition files has its own acceptance criteria.
