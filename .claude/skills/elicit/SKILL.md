---
name: elicit
description: Build an Allium specification through guided conversation. Use when the user wants to "build a spec", "elicit requirements", "capture domain behaviour", "specify a feature", or is describing functionality they want to build.
allowed-tools: Read, Bash, Grep, Glob, Write
argument-hint: [topic or feature name]
---

# Elicit

Build Allium specifications through structured conversation. The goal is to surface ambiguities and produce a spec that captures what software does without prescribing implementation.

## Methodology

### Phase 1: Scope definition

Questions to ask:
1. "What is this system fundamentally about? In one sentence?"
2. "Where does this system start and end? What's in scope vs out?"
3. "Who are the users? Are there different roles?"
4. "What are the main things being managed — the nouns?"
5. "Are there existing systems this integrates with?"

**Outputs**: Actors, core entities, boundary decisions, one-sentence description.

### Phase 2: Happy path flow

1. "Walk me through a typical [X] from start to finish"
2. "What happens first? Then what?"
3. "What triggers this? A user action? Time passing?"
4. "What changes when that happens?"
5. "Who needs to know when this happens?"

**Technique**: Follow one entity through its lifecycle. Map state machines for key entities.

**Watch for**: Jumping to edge cases too early. Implementation details creeping in.

### Phase 3: Edge cases and errors

1. "What if [actor] doesn't respond?"
2. "What if [condition] isn't met?"
3. "What if this happens twice? Or in the wrong order?"
4. "How long should we wait before [action]?"
5. "When should a human be alerted?"

**Outputs**: Timeout rules, retry logic, error states, recovery paths.

### Phase 4: Refinement and validation

1. Review entity definitions for completeness
2. Check for open questions and deferred specs
3. Confirm external boundaries

**Validation step**: After producing the `.allium` file, generate `.allium.json` and run:

```bash
bin/allium-check --format json <file>.allium.json
```

Use validation errors as conversation prompts:
- "Entity 'X' has unreachable status value 'archived' — is that intentional?"
- "Rule 'Y' references entity 'Z' which isn't declared — should we add it?"
- "Derived value 'is_ready' creates a circular dependency — which direction should we break?"

If `allium-check` is not available, suggest: `go build -o bin/allium-check ./cmd/allium-check`

## Elicitation principles

- **Ask one question at a time** — don't overwhelm
- **Work through implications** — "What happens then? And then?"
- **Distinguish product from implementation** — redirect "the API returns 404" to "the user is informed it's not found"
- **Surface ambiguity explicitly** — record open questions rather than assume
- **Use concrete examples** — "Let's say Alice is a candidate..."
- **Know when to stop** — defer complex algorithms to detailed specs

## Common traps

- **"Obviously" trap**: Probe when people say "obviously"
- **Edge case spiral**: Note edge cases, stay on happy path first
- **Technical solution trap**: Redirect to outcomes, not mechanisms
- **Vague agreement trap**: Don't accept "yes" without specifics
- **Equivalent terms trap**: Pick one term, don't note "also known as"

## Output format

Write the `.allium` file with version marker as first line:

```
-- allium: 1
-- feature-name.allium
```

Document scope at top. Include `open_question` for unresolved decisions.

## References

For the full elicitation guide with session structure and detailed examples, see [skills/elicit/SKILL.md](../../skills/elicit/SKILL.md).

For language syntax, see [references/language-reference.md](../../references/language-reference.md).

For patterns, see [references/patterns.md](../../references/patterns.md).
