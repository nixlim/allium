---
name: validate
description: Validate an Allium spec file (.allium or .allium.json) against the JSON Schema and 35 semantic rules. Use when the user wants to "validate a spec", "check for errors", "run allium-check", or "find problems in a spec".
allowed-tools: Read, Bash, Grep, Glob
argument-hint: [file.allium.json]
---

# Validate

Validates Allium specification files against the JSON Schema and 35 semantic analysis rules, then applies LLM guidance checks for naming quality and completeness.

## Prerequisites

The `allium-check` binary must be built. If not found, build it:

```bash
go build -o bin/allium-check ./cmd/allium-check
```

## Step 1: Prepare the input file

If the user provides a `.allium` file (human-readable syntax), convert it to `.allium.json` first. The JSON format is defined by the schema at `schemas/v1/allium-spec.json`. Place the generated `.allium.json` alongside the `.allium` file with the same base name.

If the user provides a `.allium.json` file directly, skip this step.

If no argument is provided (`$ARGUMENTS` is empty), look for `.allium.json` files in the current directory and ask the user which to validate.

## Step 2: Run allium-check

```bash
bin/allium-check --format json $ARGUMENTS
```

| Exit code | Meaning |
|-----------|---------|
| 0 | No errors (warnings may be present) |
| 1 | Validation errors found |
| 2 | Input error (file not found, invalid JSON) |

If exit code is 2, report the error directly and stop.

If the binary is not found:

> `allium-check` binary not found. Build it with:
> ```
> go build -o bin/allium-check ./cmd/allium-check
> ```

## Step 3: Parse and present results

Parse the JSON output. For each error, present:

1. **Rule ID and message** as reported
2. **Location** translated to human-readable form (e.g., "in entity 'User', field 'email'" not raw JSON path)
3. **Explanation** of what the rule checks
4. **Suggested fix** based on context

### Error explanation guide

| Rule | What it checks | Common fix |
|------|---------------|-----------|
| SCHEMA | JSON Schema structural validation | Fix the JSON structure to match the schema |
| RULE-01 | Entity references resolve | Declare the missing entity or fix the typo |
| RULE-03 | Relationship targets exist | Add the target entity or correct the name |
| RULE-06 | Rules sharing triggers have compatible params | Fix parameter lists to match |
| RULE-07 | All status enum values are reachable | Add a transition path to the unreachable value |
| RULE-08 | Non-terminal states have outgoing transitions | Add a transition or mark as terminal |
| RULE-09 | Status assignments use declared enum values | Fix the value or add it to the enum |
| RULE-10 | No circular derivation chains | Break the cycle |
| RULE-11 | Identifiers in scope | Declare the identifier or fix the reference |
| RULE-12 | Type compatibility in expressions | Fix types to be compatible |
| RULE-13 | Collection ops have explicit lambda params | Add the lambda parameter |
| RULE-14 | Inline enum comparison rules | Don't compare different inline enums |
| RULE-16 | Discriminator variants have declarations | Add the variant declaration |
| RULE-17 | Variants listed in base entity discriminator | Add variant to discriminator or fix base_entity |
| RULE-18 | Variant field access is type-guarded | Add type guard before accessing variant fields |
| RULE-19 | Creation uses variant name when discriminator exists | Use variant name instead of base entity |
| RULE-22 | Given binding type references exist | Fix the type reference |
| RULE-23 | Given binding names are unique | Rename the duplicate |
| RULE-26 | Config parameter names are unique | Rename the duplicate |
| RULE-27 | Config references resolve | Declare the config parameter or fix the name |
| RULE-28 | Surface facing type exists | Fix the type reference |
| RULE-29 | Surface exposes paths are reachable | Fix the field path |
| RULE-30 | Surface provides triggers exist | Fix the trigger name |
| RULE-31 | Surface related references exist | Fix the surface name |
| RULE-32 | Surface bindings are used | Reference the binding or remove it |
| RULE-33 | When conditions reference reachable fields | Fix the field reference |
| RULE-34 | for_each targets collection types | Use a collection-typed field |
| RULE-35 | Use declaration imports resolve | Fix the import reference |

### Warning guide

| Warning | What it checks | Suggestion |
|---------|---------------|-----------|
| WARN-01 | External entities without governing spec | Add a `use` declaration |
| WARN-02 | Open questions present | Resolve before finalizing |
| WARN-03 | Deferred with no location hint | Add a location hint |
| WARN-04 | Unused entity/field | Remove or reference it |
| WARN-05 | Contradictory requires | Fix the conditions |
| WARN-06 | Temporal trigger without re-firing guard | Add a `requires` guard |
| WARN-09 | Unused actor | Remove or reference in a surface |
| WARN-12 | Overlapping preconditions | Add distinguishing requires |
| WARN-14 | Trivial actor identified_by | Make the condition meaningful |
| WARN-15 | All-conditional ensures with empty path | Add a default ensures path |
| WARN-16 | Temporal trigger on optional field | Guard against null |
| WARN-17 | Raw entity type when actors available | Use an actor type instead |
| WARN-18 | transitions_to fires on creation value | Use state_becomes or add guard |
| WARN-19 | Multiple identical inline enums | Extract to a named enum |

## Step 4: LLM guidance checks

If `allium-check` returns clean (no errors), run these additional checks:

### Naming quality
- **Vague names**: "DoThing", "HandleStuff", "ProcessData" — suggest domain-specific alternatives
- **Implementation-leaking names**: "SaveToDatabase", "CallAPI" — redirect to domain behaviour
- **Inconsistent conventions**: mixing naming styles

### Spec completeness
- **Temporal rules**: Entities with `Timestamp` fields (`expires_at`, `deadline`) but no temporal trigger
- **Notification gaps**: State transitions without notifications
- **Missing error paths**: Rules with `requires` but no failure handling
- **Orphaned entities**: Entities never referenced in rules or surfaces

### Abstraction level
- **Too concrete**: References to technologies, database types, API endpoints
- **Too abstract**: Entities with no fields, rules with no ensures

## Step 5: Present summary

```
Validation complete: <file>

Schema:    PASS/FAIL
Errors:    N
Warnings:  N
Guidance:  N suggestions

[errors with fixes]
[warnings]
[guidance suggestions]
```

If all clean:

> Validation complete: spec is clean. No errors, no warnings, no guidance issues found.
