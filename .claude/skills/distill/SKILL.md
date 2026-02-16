---
name: distill
description: Extract an Allium specification from existing code. Use when the user has code and wants to "extract a spec", "distil behaviour from code", "reverse engineer a specification", or produce an Allium spec from an existing codebase.
allowed-tools: Read, Bash, Grep, Glob, Write
argument-hint: [path/to/code]
---

# Distill

Extract Allium specifications from existing codebases. The core challenge is finding the right level of abstraction — filtering out implementation details to capture domain-level behaviour.

## Scoping

Before diving into code, establish boundaries:

1. **"What subset of this codebase are we specifying?"** — Clarify which service or domain.
2. **"Is there code we should exclude?"** — Legacy, infrastructure, deprecated, experimental.
3. **The "Would we rebuild this?" test** — If rebuilt from scratch, would this be in requirements?

## Process

### Step 1: Map the territory

Identify:
- **Entry points**: API routes, CLI commands, message handlers, scheduled jobs
- **Domain models**: Usually in `models/`, `entities/`, `domain/`
- **Business logic**: Services, use cases, handlers
- **External integrations**: Third-party APIs, webhooks

### Step 2: Extract entity states

Find enum fields, status columns, constants. Convert to Allium inline enums.

### Step 3: Extract transitions

Find where status changes happen. Map code patterns to spec patterns:

| Code pattern | Spec pattern |
|--------------|--------------|
| `if x.status != 'pending': raise` | `requires: x.status = pending` |
| `x.status = 'accepted'` | `ensures: x.status = accepted` |
| `Model.create(...)` | `ensures: Model.created(...)` |
| `send_email(...)` | `ensures: Email.created(...)` |

### Step 4: Find temporal triggers

Look for scheduled jobs and time-based logic. Convert to temporal triggers with re-firing guards.

### Step 5: Identify external boundaries

Third-party APIs, webhook handlers, imported data → external entities.

### Step 6: Abstract away implementation

Remove: database types, ORM syntax, HTTP details, framework concepts, infrastructure. Replace FKs with relationships. Remove tokens/secrets. Use domain Duration not timedelta.

### Step 7: Write the `.allium` file

Assemble the complete spec. The version marker (`-- allium: 1`) must be the first line.

### Step 8: Generate JSON and validate

1. Convert `.allium` to `.allium.json` (JSON AST matching `schemas/v1/allium-spec.json`)
2. Run: `bin/allium-check --format json <file>.allium.json`
3. Fix any validation errors before presenting the final spec
4. If `allium-check` is not available, note this and suggest: `go build -o bin/allium-check ./cmd/allium-check`

## Abstraction checklist

Before finalising:

- [ ] No database column types
- [ ] No ORM or query syntax
- [ ] No HTTP status codes or API paths
- [ ] No framework-specific concepts
- [ ] No programming language types
- [ ] No infrastructure (Redis, Kafka, S3)
- [ ] Foreign keys replaced with relationships
- [ ] Tokens/secrets removed
- [ ] Timestamps use domain Duration

## References

For full distillation guidance including the "Why test", "Could it be different? test", library spec detection, and worked examples, see [skills/distill/SKILL.md](../../skills/distill/SKILL.md).

For language syntax, see [references/language-reference.md](../../references/language-reference.md).
