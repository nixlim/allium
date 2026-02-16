---
name: validate
description: This skill should be used when the user wants to "validate an allium spec", "check a spec for errors", "run allium-check", "find problems in a spec", or has a .allium or .allium.json file they want to verify against the schema and semantic rules.
---

# Validate

This skill validates Allium specification files against the JSON Schema and 35 semantic analysis rules. It runs the deterministic `allium-check` CLI and then applies LLM guidance checks for naming quality and completeness.

## Prerequisites

The `allium-check` binary must be built and available. If it is not found, build it:

```bash
go build -o bin/allium-check ./cmd/allium-check
```

Then ensure `bin/` is in your PATH or use the full path to the binary.

## Step 1: Prepare the input file

If the user provides a `.allium` file (human-readable syntax), convert it to `.allium.json` first. The JSON format is defined by the schema at `schemas/v1/allium-spec.json`.

Place the generated `.allium.json` file alongside the `.allium` file (same directory, same base name).

If the user provides a `.allium.json` file directly, skip this step.

## Step 2: Run allium-check

Invoke the CLI with JSON output for structured results:

```bash
allium-check --format json <file>.allium.json
```

Capture both stdout (the JSON report) and the exit code:

| Exit code | Meaning |
|-----------|---------|
| 0 | No errors (warnings may be present) |
| 1 | Validation errors found |
| 2 | Input error (file not found, invalid JSON, bad flags) |

### Handling exit code 2

If exit code is 2, the file could not be parsed or found. Report the error directly to the user without proceeding to further checks.

### Handling missing binary

If the `allium-check` command is not found:

> `allium-check` binary not found. Build it with:
> ```
> go build -o bin/allium-check ./cmd/allium-check
> ```

Then retry after building.

## Step 3: Parse and present results

Parse the JSON output. The report structure is:

```json
{
  "file": "path/to/spec.allium.json",
  "schema_valid": true,
  "errors": [
    {
      "rule": "RULE-01",
      "severity": "error",
      "message": "Entity 'Foo' referenced but not declared",
      "location": { "file": "...", "path": "$.entities[0].fields[1].type" }
    }
  ],
  "warnings": [
    {
      "rule": "WARN-01",
      "severity": "warning",
      "message": "External entity 'Email' has no governing spec",
      "location": { "file": "...", "path": "$.external_entities[0]" }
    }
  ],
  "summary": { "error_count": 0, "warning_count": 0 }
}
```

For each error, present:

1. **Rule ID and message** as reported
2. **Location** translated to a human-readable pointer (e.g., "in entity 'User', field 'email'" rather than raw JSON path)
3. **Explanation** of what the rule checks and why it matters
4. **Suggested fix** based on the error context

### Error explanation guide

| Rule | What it checks | Common fix |
|------|---------------|-----------|
| SCHEMA | JSON Schema structural validation | Fix the JSON structure to match the schema |
| RULE-01 | Entity references resolve to declared entities | Declare the missing entity or fix the typo |
| RULE-03 | Relationship targets exist | Add the target entity or correct the name |
| RULE-06 | Entity, rule, surface names are unique | Rename the duplicate |
| RULE-07 | All status enum values are reachable | Add a creation or transition path to the unreachable value |
| RULE-08 | Non-terminal states have outgoing transitions | Add a transition or mark as intentionally terminal |
| RULE-09 | Status assignments use declared enum values | Fix the value to match the enum or add it to the enum |
| RULE-10 | Circular derivation chains | Break the cycle by removing one derived dependency |
| RULE-11 | Identifiers in scope | Declare the identifier or fix the reference |
| RULE-12 | Boolean context expressions | Ensure the expression evaluates to a boolean |
| RULE-13 | Temporal rule guards against re-firing | Add a `requires` clause that prevents re-triggering |
| RULE-14 | Expression type compatibility | Fix the types to be compatible |
| RULE-16 | Sum type discriminator values are PascalCase | Capitalize variant names |
| RULE-17 | Sum type variants declared for each discriminator value | Add missing variant declarations |
| RULE-18 | Variant field access guarded by type check | Add an `if` guard narrowing to the variant type |
| RULE-19 | Variant base entity has discriminator field | Add a discriminator field to the base entity |
| RULE-22 | Given bindings reference valid types | Fix the type reference |
| RULE-23 | Given binding names are unique | Rename the duplicate binding |
| RULE-26 | Config parameter names are unique | Rename the duplicate parameter |
| RULE-27 | Config references resolve | Declare the config parameter or fix the name |
| RULE-28 | Surface facing type exists | Fix the type reference |
| RULE-29 | Surface exposes paths are reachable | Fix the field path or add the field |
| RULE-30 | Surface provides triggers exist | Fix the trigger name |
| RULE-31 | Surface related references exist | Fix the surface name |
| RULE-32 | Surface bindings are used | Reference the binding in the surface body or remove it |
| RULE-33 | Surface when conditions reference reachable fields | Fix the field reference |
| RULE-34 | Surface for_each targets collection types | Change to a collection-typed field |
| RULE-35 | Use declaration imports resolve | Fix the import reference |

### Warning explanation guide

| Warning | What it checks | Suggestion |
|---------|---------------|-----------|
| WARN-01 | External entities without governing spec | Consider adding a `use` declaration for the external spec |
| WARN-02 | Open questions still present | Resolve or defer open questions before finalizing |
| WARN-17 | Raw entity type in surface facing when actors available | Use an actor type instead of the raw entity |

## Step 4: LLM guidance checks

If `allium-check` returns clean (no errors), run these additional checks using LLM reasoning. These are subjective quality checks that go beyond deterministic validation.

### Guidance check 1: Naming quality

Review all entity, rule, and surface names for:

- **Vague names**: "DoThing", "HandleStuff", "ProcessData", "UpdateRecord" — names that don't convey domain meaning
- **Implementation-leaking names**: "SaveToDatabase", "CallAPI", "SendHTTPRequest" — names that describe implementation rather than domain behaviour
- **Inconsistent conventions**: mixing naming styles within the same spec

For each issue found, suggest a specific improvement:

> Rule name 'DoThing' is not descriptive. Consider naming it after the business event it handles, e.g., 'CandidateAcceptsInvitation'.

### Guidance check 2: Spec completeness

Look for common missing patterns:

- **Temporal rules**: Entities with `Timestamp` fields (especially `expires_at`, `deadline`, `due_at`) but no corresponding temporal trigger rule
- **Notification gaps**: State transitions that would typically notify someone but have no notification ensures
- **Missing error paths**: Rules with `requires` clauses but no corresponding rule handling the failure case
- **Orphaned entities**: Entities declared but never referenced in any rule trigger, ensures, or surface

For each issue found, suggest what might be missing:

> Entity 'Invitation' has field 'expires_at' but no temporal rule. Consider adding a rule like 'InvitationExpires' that triggers when the deadline passes.

### Guidance check 3: Abstraction level

Check for signs of over- or under-specification:

- **Too concrete**: References to specific technologies, database types, API endpoints, HTTP methods
- **Too abstract**: Entities with no fields, rules with no requires or ensures, surfaces with no exposes

## Step 5: Present summary

Present a final summary:

```
Validation complete: <file>

Schema:    PASS/FAIL
Errors:    N
Warnings:  N
Guidance:  N suggestions

[List of errors with fixes]
[List of warnings]
[List of guidance suggestions]
```

If all checks pass with no findings:

> Validation complete: spec is clean. No errors, no warnings, no guidance issues found.
