# Plan: JSON Schema Validation for Allium

## Context

Allium is a behavioral specification language with 35 formal validation rules defined in prose but no tooling to enforce them. The distill/elicit skills generate `.allium` files that no one can verify. Adding schema validation closes this gap — giving the LLM a feedback loop: generate, validate, fix.

## Approach

**Three-layer validation within the allium repo:**

1. **JSON Schema layer** (deterministic, embedded in Go CLI) — Validates structural shape of every construct. Catches ~40% of errors immediately.
2. **Semantic validation layer** (deterministic, Go CLI) — Cross-references, state machine analysis, cycle detection, type consistency, and all 35 rules. Runs as part of the same CLI binary.
3. **LLM validation skill** — A `validate` skill that invokes the CLI and interprets results. Also provides guidance-level checks that require LLM reasoning (e.g., "is this rule name descriptive enough?").

**The Go CLI (`allium-check`) is the core deterministic validator.** It:
- Reads `.allium.json` files
- Validates against the embedded JSON Schema
- Runs all 35 semantic validation rules
- Runs all ~15 warning checks
- Outputs structured JSON or human-readable reports
- Ships as a single static binary (no dependencies)
- Can be used standalone in CI pipelines or invoked by the validate skill

The distill and elicit skills are modified to output a `.allium.json` alongside every `.allium` file, then invoke `allium-check`.

## File Structure

```
allium/
  schemas/v1/
    allium-spec.json                          # Root schema (composes all via $ref)
    definitions/
      common.json                             # Naming patterns, metadata
      field-types.json                        # FieldType discriminated union
      expressions.json                        # Expression AST (30 node types)
      entities.json                           # Entity, ExternalEntity, ValueType, Variant
      enumerations.json                       # Named enum
      rules.json                              # Rule, 7 Trigger types, EnsuresClause variants
      surfaces.json                           # Surface + all sub-clauses
      actors.json                             # Actor declarations
      config.json                             # Config block
      defaults.json                           # Default declarations
      given.json                              # Given block
      use-declarations.json                   # Use declarations
      deferred.json                           # Deferred specs
      open-questions.json                     # Open questions
    examples/
      password-auth.allium.json               # Pattern 1 converted as reference
  cmd/
    allium-check/                             # Go CLI binary
      main.go                                 # Entry point, CLI flags, output formatting
      main_test.go                            # Integration tests
  internal/
    schema/
      embed.go                                # Embeds JSON Schema files via go:embed
      validate.go                             # JSON Schema validation against embedded schemas
      validate_test.go
    ast/
      types.go                                # Go structs mirroring the JSON AST
      load.go                                 # JSON -> Go struct deserialization
      load_test.go
    check/
      checker.go                              # Orchestrates all validation passes
      checker_test.go
      structural.go                           # Rules 1-6: reference resolution, relationships
      structural_test.go
      statemachine.go                         # Rules 7-9: reachability, exits, undefined states
      statemachine_test.go
      expression.go                           # Rules 10-14: cycles, scope, types, lambdas, enums
      expression_test.go
      sumtype.go                              # Rules 15-21: discriminators, variants, guards
      sumtype_test.go
      given.go                                # Rules 22-24: valid types, unique names, resolution
      given_test.go
      config.go                               # Rules 25-27: types, uniqueness, references
      config_test.go
      surface.go                              # Rules 28-35: facing, exposes, provides, related
      surface_test.go
      warnings.go                             # ~15 warning-level checks
      warnings_test.go
    report/
      report.go                               # Structured report types (Error, Warning, Report)
      format.go                               # Output formatters (JSON, text, SARIF)
      format_test.go
  go.mod
  go.sum
  validation/
    VALIDATION-RULES.md                       # Complete rule-to-check mapping
    rules/
      structural.md                           # Rules 1-6
      state-machine.md                        # Rules 7-9
      expression.md                           # Rules 10-14
      sum-type.md                             # Rules 15-21
      given.md                                # Rules 22-24
      config.md                               # Rules 25-27
      surface.md                              # Rules 28-35
    warnings.md                               # ~15 warning checks
  skills/validate/
    SKILL.md                                  # The validate skill (invokes allium-check)
    references/
      validation-checklist.md                 # Step-by-step LLM checklist
      json-ast-guide.md                       # .allium -> JSON mapping guide
```

## Key Design Decisions

### Expression representation: pragmatic hybrid

- ~80% of expressions get a typed AST (field access, comparisons, boolean, arithmetic, collection ops, existence, literals)
- Complex/ambiguous expressions use a `"kind": "raw"` escape hatch with source string and type hint
- This avoids schema explosion while still enabling structural validation of common patterns

### Schema organization: modular with $ref

- 14 definition files composed by the root schema
- Each construct schema testable and reviewable independently
- Versioned under `schemas/v1/` for future language evolution

### JSON AST lives alongside .allium

- `.allium` remains the primary human-readable artifact
- `.allium.json` is derived, machine-readable, always regenerable
- Both files coexist in the repo

## JSON AST Structure

### Top-level document

```json
{
  "$schema": "https://allium-lang.org/schemas/v1/allium-spec.json",
  "version": 1,
  "file": "interview-scheduling.allium",
  "metadata": {
    "scope": "Interview scheduling for hiring pipeline",
    "includes": ["Candidacy", "Interview", "InterviewSlot"],
    "excludes": ["Authentication", "Reporting"]
  },
  "use_declarations": [],
  "given": null,
  "external_entities": [],
  "value_types": [],
  "enumerations": [],
  "entities": [],
  "variants": [],
  "config": [],
  "defaults": [],
  "rules": [],
  "actors": [],
  "surfaces": [],
  "deferred_specs": [],
  "open_questions": []
}
```

### FieldType (discriminated by `kind`)

```json
// Primitive
{ "kind": "primitive", "primitive": "String" | "Integer" | "Decimal" | "Boolean" | "Timestamp" | "Duration" }

// Compound
{ "kind": "compound", "compound": "Set" | "List", "inner": <FieldType> }

// Optional
{ "kind": "optional", "inner": <FieldType> }

// Entity reference
{ "kind": "entity_ref", "entity": "Candidate", "namespace": null }

// Inline enum
{ "kind": "inline_enum", "values": ["pending", "active", "completed"] }

// Named enum
{ "kind": "named_enum", "enum_name": "Recommendation" }

// Discriminator (sum type)
{ "kind": "discriminator", "variants": ["Branch", "Leaf"] }
```

### Entity (internal)

```json
{
  "name": "Candidacy",
  "kind": "internal",
  "fields": [
    { "name": "candidate", "type": { "kind": "entity_ref", "entity": "Candidate" } },
    { "name": "status", "type": { "kind": "inline_enum", "values": ["pending", "active", "completed", "cancelled"] } },
    { "name": "retry_count", "type": { "kind": "primitive", "primitive": "Integer" } }
  ],
  "relationships": [
    {
      "name": "invitation",
      "target_entity": "Invitation",
      "cardinality": "one",
      "backreference": { "field": "candidacy", "value": "this" }
    },
    {
      "name": "slots",
      "target_entity": "InterviewSlot",
      "cardinality": "many",
      "backreference": { "field": "candidacy", "value": "this" }
    }
  ],
  "projections": [
    {
      "name": "confirmed_slots",
      "source": "slots",
      "filter": { "kind": "comparison", "left": { "kind": "field_access", "path": ["status"] }, "op": "=", "right": { "kind": "enum_literal", "value": "confirmed" } },
      "map_field": null
    }
  ],
  "derived_values": [
    {
      "name": "is_ready",
      "parameters": [],
      "expression": { "kind": "comparison", "left": { "kind": "field_access", "path": ["confirmed_slots", "count"] }, "op": ">=", "right": { "kind": "literal", "type": "Integer", "value": 3 } }
    }
  ],
  "discriminator": null
}
```

### Trigger types (discriminated by `kind`)

```json
// External stimulus
{ "kind": "external_stimulus", "name": "CandidateSelectsSlot", "parameters": [
    { "name": "invitation", "type": null, "optional": false },
    { "name": "slot", "type": null, "optional": false }
  ]
}

// State transition
{ "kind": "state_transition", "binding": "interview", "entity": "Interview", "field": "status", "target_value": "scheduled" }

// State becomes
{ "kind": "state_becomes", "binding": "interview", "entity": "Interview", "field": "status", "target_value": "scheduled" }

// Temporal
{ "kind": "temporal", "binding": "invitation", "entity": "Invitation", "expression": <Expression> }

// Derived condition
{ "kind": "derived_condition", "binding": "interview", "entity": "Interview", "derived_field": "all_feedback_in" }

// Entity creation
{ "kind": "entity_creation", "binding": "batch", "entity": "DigestBatch" }

// Chained
{ "kind": "chained", "trigger_name": "AllConfirmationsResolved", "parameters": [
    { "name": "candidacy", "type": null, "optional": false }
  ]
}
```

### Rule

```json
{
  "name": "InvitationExpires",
  "trigger": {
    "kind": "temporal",
    "binding": "invitation",
    "entity": "Invitation",
    "expression": { "kind": "comparison", "left": { "kind": "field_access", "path": ["expires_at"] }, "op": "<=", "right": { "kind": "keyword", "keyword": "now" } }
  },
  "for_clause": null,
  "let_bindings": [],
  "requires": [
    { "expression": { "kind": "comparison", "left": { "kind": "field_access", "path": ["invitation", "status"] }, "op": "=", "right": { "kind": "enum_literal", "value": "pending" } } }
  ],
  "ensures": [
    {
      "kind": "state_change",
      "target": { "kind": "field_access", "path": ["invitation", "status"] },
      "value": { "kind": "enum_literal", "value": "expired" }
    }
  ]
}
```

### EnsuresClause types (discriminated by `kind`)

```json
// State change
{ "kind": "state_change", "target": <Expression>, "value": <Expression> }

// Entity creation
{ "kind": "entity_creation", "entity": "Interview", "namespace": null, "arguments": { "candidacy": <Expression>, "slot": <Expression> }, "let_binding": null }

// Trigger emission
{ "kind": "trigger_emission", "trigger_name": "CandidateInformed", "arguments": { "candidate": <Expression>, "about": <Expression> } }

// Entity removal
{ "kind": "entity_removal", "target": <Expression> }

// Conditional
{ "kind": "conditional", "condition": <Expression>, "then_body": [<EnsuresClause>], "else_body": [<EnsuresClause>] }

// Iteration
{ "kind": "iteration", "binding": "s", "collection": <Expression>, "filter": null, "body": [<EnsuresClause>] }

// Let binding (within ensures)
{ "kind": "let_binding", "name": "slot", "expression": <Expression> }

// Set mutation
{ "kind": "set_mutation", "target": <Expression>, "operation": "add" | "remove", "value": <Expression> }
```

### Expression AST (discriminated by `kind`, 30 variants)

```json
// Field access / navigation
{ "kind": "field_access", "path": ["interview", "candidacy", "candidate", "email"] }

// Optional navigation
{ "kind": "optional_access", "base": <Expression>, "field": "effective_permissions" }

// Null coalescing
{ "kind": "null_coalesce", "left": <Expression>, "right": <Expression> }

// Comparison
{ "kind": "comparison", "left": <Expression>, "op": "=" | "!=" | "<" | "<=" | ">" | ">=", "right": <Expression> }

// Boolean logic
{ "kind": "boolean", "op": "and" | "or", "left": <Expression>, "right": <Expression> }
{ "kind": "not", "operand": <Expression> }

// Arithmetic
{ "kind": "arithmetic", "op": "+" | "-" | "*" | "/", "left": <Expression>, "right": <Expression> }

// Membership
{ "kind": "in", "element": <Expression>, "collection": <Expression> }
{ "kind": "not_in", "element": <Expression>, "collection": <Expression> }

// Collection operations
{ "kind": "count", "collection": <Expression> }
{ "kind": "any", "collection": <Expression>, "lambda_param": "i", "predicate": <Expression> }
{ "kind": "all", "collection": <Expression>, "lambda_param": "c", "predicate": <Expression> }
{ "kind": "first", "collection": <Expression> }
{ "kind": "last", "collection": <Expression> }

// Existence
{ "kind": "exists", "target": <Expression> }
{ "kind": "not_exists", "target": <Expression> }

// Join lookup
{ "kind": "join_lookup", "entity": "SlotConfirmation", "fields": { "slot": <Expression>, "interviewer": <Expression> } }

// Literals
{ "kind": "literal", "type": "String" | "Integer" | "Decimal" | "Boolean", "value": <any> }
{ "kind": "enum_literal", "value": "pending" }
{ "kind": "set_literal", "values": [<Expression>] }
{ "kind": "object_literal", "fields": { "key": <Expression> } }
{ "kind": "duration_literal", "value": 24, "unit": "hours" }

// Keywords
{ "kind": "keyword", "keyword": "now" | "this" | "null" | "true" | "false" | "within" }

// Discard
{ "kind": "discard" }

// Black box function
{ "kind": "black_box", "name": "hash", "arguments": [<Expression>] }

// Deferred spec invocation
{ "kind": "deferred_invocation", "path": "InterviewerMatching.suggest", "arguments": [<Expression>] }

// Projection expression
{ "kind": "projection", "source": <Expression>, "filter": <Expression>, "map_field": null }

// Conditional expression (inline if/else)
{ "kind": "conditional_expr", "condition": <Expression>, "then_value": <Expression>, "else_value": <Expression> }

// Entity collection
{ "kind": "entity_collection", "entity": "Users" }

// Raw (escape hatch for complex expressions)
{ "kind": "raw", "source": "complex expression text", "hint": "boolean" | "value" | "collection" }
```

### Surface

```json
{
  "name": "InterviewerDashboard",
  "facing": { "binding": "viewer", "type": "Interviewer", "is_actor": true },
  "context": {
    "binding": "assignment",
    "entity": "SlotConfirmation",
    "filter": <Expression>
  },
  "let_bindings": [],
  "exposes": [
    { "kind": "field", "expression": { "kind": "field_access", "path": ["assignment", "slot", "time"] } },
    { "kind": "field", "expression": { "kind": "field_access", "path": ["assignment", "status"] } }
  ],
  "provides": [
    {
      "trigger_name": "InterviewerConfirmsSlot",
      "arguments": [<Expression>],
      "when_condition": <Expression>
    }
  ],
  "guarantee": null,
  "guidance": null,
  "related": [
    {
      "surface_name": "InterviewDetail",
      "context_expression": <Expression>,
      "when_condition": <Expression>
    }
  ],
  "timeout": [
    {
      "rule_name": "InvitationExpires",
      "when_condition": <Expression>
    }
  ]
}
```

### Actor

```json
{
  "name": "WorkspaceAdmin",
  "within": "Workspace",
  "identified_by": {
    "entity": "User",
    "condition": <Expression>
  }
}
```

## Validation Rule Mapping

### Schema-enforceable (7 of 35 rules)

| Rule | Schema Enforcement |
|------|--------------------|
| 2 (fields have types) | `Field` requires `name` + `type`, type must match `FieldType` schema |
| 4 (rules have trigger + ensures) | `Rule` requires both, `ensures` has `minItems: 1` |
| 5 (trigger types valid) | `Trigger` is `oneOf` with 7 discriminated variants |
| 15 (discriminators capitalized) | `FieldType.discriminator.variants` uses PascalCase pattern |
| 20 (discriminator names user-defined) | No reserved name; `snake_case_name` pattern |
| 21 (variant keyword required) | Variants in separate array with `required: ["name", "base_entity"]` |
| 25 (config types + defaults) | `ConfigParameter` requires name, type, and default_value |

### Semantic validation (28 rules, 7 check passes)

**Pass 1 — Reference checks (Rules 1, 3, 22, 27, 28, 30, 31, 35):**
Walk all entity_ref types, relationship targets, trigger entities, surface types, provides triggers, related surfaces, timeout rules. Verify each name resolves to a declared entity, actor, rule, or import.

**Pass 2 — Uniqueness checks (Rules 6, 23, 26):**
- Rule 6: Rules sharing a trigger name must have same parameter count and positional types
- Rule 23: No duplicate names in `given` bindings
- Rule 26: No duplicate names in config parameters

**Pass 3 — State machine analysis (Rules 7-9):**
- Rule 7: For each entity status field, collect all ensures that set it. Build reachability graph from creation values. Warn for unreachable values.
- Rule 8: Non-terminal status values must have at least one outgoing transition rule.
- Rule 9: Ensures that set an enum field must use a value declared in the enum.

**Pass 4 — Expression analysis (Rules 10-14):**
- Rule 10: Build dependency graph of derived values; detect cycles.
- Rule 11: Walk each rule tracking scope (trigger bindings, for-clause, let, given, defaults). Every field_access root must be in scope.
- Rule 12: Both sides of comparisons must be compatible types. Arithmetic operands must be numeric or Timestamp/Duration.
- Rule 13: `any`/`all` expressions must use explicit lambda parameters.
- Rule 14: Inline enum fields cannot be compared with each other. Only named enums of the same type may be compared.

**Pass 5 — Sum type analysis (Rules 16-19):**
- Rule 16: Every discriminator variant name must have a corresponding `variant` declaration.
- Rule 17: Every variant must be listed in its base entity's discriminator.
- Rule 18: Variant-specific field access must be within a type guard (requires or if branch checking discriminator = VariantName).
- Rule 19: Entity creation via `.created()` must use variant name, not base entity name, when a discriminator exists.

**Pass 6 — Surface analysis (Rules 29, 32-34):**
- Rule 29: All field paths in `exposes` must be reachable from facing, context, or let bindings.
- Rule 32: Bindings from facing and context must be used somewhere in the surface.
- Rule 33: `when` conditions must reference valid fields reachable from party or context bindings.
- Rule 34: `for` iterations must iterate over collection-typed fields/bindings.

**Pass 7 — Warning pass (~15 warnings):**

| Warning | Check |
|---------|-------|
| External entities without governing spec | Not referenced by any `use` declaration's imported types |
| Open questions present | Any entry in `open_questions` |
| Deferred specs without location hints | `location_hint` is null or empty |
| Unused entities or fields | Not referenced by any rule, surface, relationship, or entity |
| Rules that can never fire | Contradictory requires (e.g., `status = A and status = B`) |
| Temporal rules without guards | Temporal trigger without a requires that prevents re-firing |
| Surfaces referencing unused fields | Fields in `exposes` not used by any rule |
| Provides with impossible when conditions | `when` conditions that are always false |
| Unused actor declarations | Not referenced in any surface `facing` clause |
| Sibling rule entity creation guards | Creates entity for parent without guarding against duplicates |
| Surface provides weaker than rule requires | `when` condition is strictly weaker than corresponding rule's requires |
| Overlapping preconditions | Same trigger, requires clauses could be simultaneously true |
| Parameterised derived scope violation | Derived values with params referencing fields outside entity |
| Trivial actor identified_by | Condition always true or always false |
| All-conditional ensures with empty paths | All ensures conditional, at least one path produces no effects |
| Temporal triggers on optional fields | Field is `T?`, trigger won't fire when absent |
| Raw entity type with actors available | Surface uses raw entity in facing when actors exist for it |
| transitions_to on creation values | Trigger on value entities can be created with |
| Multiple identical inline enums | Same entity, multiple fields with identical inline enum literals |

## Go CLI: `allium-check`

### Overview

A single static Go binary that provides fully deterministic validation. No runtime dependencies. Embeds the JSON Schema files via `go:embed`. Ships as pre-built binaries for linux/amd64, darwin/amd64, darwin/arm64.

### CLI Interface

```
Usage: allium-check [flags] <file.allium.json> [<file2.allium.json> ...]

Flags:
  -f, --format    Output format: text (default), json, sarif
  -w, --warnings  Include warnings (default: true)
  -q, --quiet     Only output errors, suppress warnings
  -s, --strict    Treat warnings as errors (exit code 1 if any warnings)
  --schema-only   Only run JSON Schema validation, skip semantic checks
  --rules         Comma-separated rule IDs to check (e.g., "1,3,7-9,28")
  --version       Print version and exit
```

### Exit Codes

- `0` — No errors (warnings may be present)
- `1` — Validation errors found
- `2` — Input/parsing errors (file not found, invalid JSON)

### Output Format (JSON)

```json
{
  "file": "interview-scheduling.allium.json",
  "schema_valid": true,
  "errors": [
    {
      "rule": "RULE-01",
      "severity": "error",
      "message": "Entity 'FooBar' referenced in Candidacy.foo but not declared",
      "location": { "entity": "Candidacy", "field": "foo" }
    }
  ],
  "warnings": [
    {
      "rule": "WARN-TEMPORAL",
      "severity": "warning",
      "message": "Rule 'InvitationExpires' has temporal trigger without re-firing guard",
      "location": { "rule": "InvitationExpires" }
    }
  ],
  "summary": {
    "errors": 1,
    "warnings": 1,
    "entities": 12,
    "rules": 8,
    "surfaces": 3
  }
}
```

### Architecture

```
cmd/allium-check/main.go
  │
  ├── internal/schema/          # Layer 1: JSON Schema validation
  │     embed.go                  Embeds schemas/v1/**/*.json via go:embed
  │     validate.go               Uses github.com/santhosh-tekuri/jsonschema/v6
  │
  ├── internal/ast/             # JSON -> Go structs
  │     types.go                  AlliumSpec, Entity, Rule, Surface, Expression, etc.
  │     load.go                   json.Unmarshal into typed structs
  │
  ├── internal/check/           # Layer 2: Semantic validation
  │     checker.go                Orchestrates passes, builds symbol table
  │     structural.go             Rules 1-6
  │     statemachine.go           Rules 7-9
  │     expression.go             Rules 10-14
  │     sumtype.go                Rules 15-21
  │     given.go                  Rules 22-24
  │     config.go                 Rules 25-27
  │     surface.go                Rules 28-35
  │     warnings.go               ~15 warning checks
  │
  └── internal/report/          # Output formatting
        report.go                 Error/Warning/Report types
        format.go                 text, json, sarif formatters
```

### Key Implementation Details

**Symbol Table** (`checker.go`):
The checker's first pass builds a symbol table collecting all:
- Entity names (internal, external, imported)
- Value type names
- Named enum names + values
- Variant names + base entity mapping
- Rule names + trigger names
- Actor names
- Surface names
- Config parameter names
- Given binding names
- Default instance names

Every subsequent pass queries this table for reference resolution.

**State Machine Graph** (`statemachine.go`):
For each entity with an enum-typed status field:
1. Collect all creation points (`.created()` in ensures) → initial states
2. Collect all state change assignments → transitions
3. Build directed graph: state → state
4. BFS from initial states → mark reachable
5. Report unreachable states (Rule 7)
6. Report states with no outgoing edges that aren't terminal (Rule 8)
7. Report assignments to undeclared values (Rule 9)

**Cycle Detection** (`expression.go`):
For derived value dependency analysis (Rule 10):
1. Build adjacency list: derived_value → set of derived_values it references
2. Run Tarjan's SCC algorithm
3. Any SCC of size > 1 is a cycle → error

**Type Guards** (`sumtype.go`):
For Rule 18 (variant fields only accessible within guards):
1. Walk each rule's expression tree
2. Track "active guards" — conditions that narrow a binding to a specific variant
3. When a field_access reaches a variant-specific field, check if the appropriate guard is active
4. Guards come from: `requires` clauses comparing discriminator to variant name, `if` branches with same comparison

### Dependencies

Minimal:
- `github.com/santhosh-tekuri/jsonschema/v6` — JSON Schema validation (well-maintained, supports draft 2020-12)
- Standard library only for everything else

### Build & Release

```makefile
# Makefile
build:
	go build -o bin/allium-check ./cmd/allium-check

test:
	go test ./...

release:
	GOOS=linux GOARCH=amd64 go build -o bin/allium-check-linux-amd64 ./cmd/allium-check
	GOOS=darwin GOARCH=amd64 go build -o bin/allium-check-darwin-amd64 ./cmd/allium-check
	GOOS=darwin GOARCH=arm64 go build -o bin/allium-check-darwin-arm64 ./cmd/allium-check
```

### Testing Strategy

Each `check/*.go` file has a corresponding `*_test.go` with:
- **Valid specs**: Confirm no errors for well-formed specs
- **Targeted violations**: One test per rule, each constructing a minimal spec that violates exactly that rule
- **Edge cases**: Optional fields, empty collections, deep nesting, cross-module references

Integration test in `main_test.go`:
- Runs `allium-check` against `schemas/v1/examples/password-auth.allium.json`
- Verifies clean pass
- Runs against intentionally broken variants
- Verifies correct error codes in output

## Changes to Existing Skills

### distill/SKILL.md — Add Step 8

After "Validate with stakeholders":

> **Step 8: Generate JSON AST and validate**
>
> For each `.allium` file produced, generate a parallel `.allium.json` containing the JSON AST representation per `schemas/v1/allium-spec.json`. Invoke the validate skill. Fix any validation errors before presenting to the user.

### elicit/SKILL.md — Add validation integration

After session output, generate `.allium.json` and validate. During Phase 4 (Refinement), use validation errors as conversation prompts:
- "Entity X has a status value 'archived' that no rule ever reaches. Is that intentional?"
- "The temporal rule for invitation expiry doesn't have a guard against re-firing. What prevents it from firing repeatedly?"

### SKILL.md — Add routing table entry

```
| Validating a spec | `validate` | User wants to check a .allium file or JSON AST for correctness |
```

## Validate Skill Design

The validate skill is a thin LLM wrapper around `allium-check`. Its value is:
1. **Generating the JSON AST** from `.allium` files (the LLM parses Allium syntax → JSON)
2. **Invoking `allium-check`** on the generated JSON
3. **Interpreting results** and suggesting fixes
4. **Guidance-level checks** that require LLM judgment (naming quality, spec completeness, domain appropriateness)

### Workflow

```
User has .allium file
  → Validate skill parses .allium → .allium.json (LLM does this)
  → Skill runs: allium-check <file>.allium.json --format json
  → Skill reads JSON output
  → If errors: presents them with suggested fixes
  → If warnings: presents them with context
  → If clean: confirms spec is valid
```

### The skill does NOT duplicate what the CLI does

The CLI handles all deterministic validation. The skill adds:
- `.allium` → `.allium.json` conversion (requires understanding Allium syntax)
- Human-readable interpretation of CLI output
- Fix suggestions ("Entity X is referenced but not declared — did you mean Y?")
- Guidance checks: "This rule name 'DoThing' is vague — consider a more descriptive name"

## Implementation Sequence

1. **Phase 1** (15 files): JSON Schema definitions — common, field-types, expressions, entities, enumerations, rules, surfaces, actors, config, defaults, given, use-declarations, deferred, open-questions, root schema

2. **Phase 2**: Go CLI foundation
   - `go.mod`, `go.sum` — module init
   - `internal/ast/types.go` — Go structs mirroring JSON AST
   - `internal/ast/load.go` — JSON deserialization
   - `internal/schema/embed.go` — Embed JSON Schema files
   - `internal/schema/validate.go` — Schema validation
   - `internal/report/report.go` — Error/Warning types
   - `internal/report/format.go` — Output formatters
   - `cmd/allium-check/main.go` — CLI entry point

3. **Phase 3**: Semantic validation passes (the bulk of the work)
   - `internal/check/checker.go` — Orchestrator + symbol table
   - `internal/check/structural.go` — Rules 1-6
   - `internal/check/statemachine.go` — Rules 7-9
   - `internal/check/expression.go` — Rules 10-14
   - `internal/check/sumtype.go` — Rules 15-21
   - `internal/check/given.go` — Rules 22-24
   - `internal/check/config.go` — Rules 25-27
   - `internal/check/surface.go` — Rules 28-35
   - `internal/check/warnings.go` — ~15 warning checks
   - Tests for each file

4. **Phase 4** (9 files): Validation documentation — VALIDATION-RULES.md, 7 rule group docs, warnings doc

5. **Phase 5** (3 files): Validate skill — SKILL.md, validation-checklist.md, json-ast-guide.md

6. **Phase 6** (3 files): Modify existing skills — distill, elicit, root SKILL.md

7. **Phase 7**: Verification
   - Convert Pattern 1 (Password Auth) to `.allium.json`
   - Run `allium-check` against it — must pass clean
   - Create intentionally broken variants — verify correct errors
   - Run `go test ./...` — all tests pass

## Naming Convention Enforcement (via JSON Schema patterns)

```json
"PascalCaseName": { "type": "string", "pattern": "^[A-Z][a-zA-Z0-9]*$" }
"snake_case_name": { "type": "string", "pattern": "^[a-z][a-z0-9_]*$" }
"QualifiedName":   { "type": "string", "pattern": "^([a-z][a-z0-9_-]*/)?[A-Z][a-zA-Z0-9]*$" }
```

- PascalCase: entity names, variant names, rule names, trigger names, actor names, surface names
- snake_case: field names, config parameters, derived values, enum literals, relationship names
