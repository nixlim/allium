# Allium

Allium is a formal language for capturing software behaviour at the domain level. This repo contains:

- **Language reference**: `references/language-reference.md`
- **Patterns library**: `references/patterns.md`
- **Validator**: Go CLI (`allium-check`) that validates `.allium.json` files against JSON Schema + 35 semantic rules

## Project structure

```
cmd/allium-check/       CLI binary (main.go)
internal/
  ast/                  Go types for the JSON AST + loader
  checker/              Orchestrates schema + semantic validation passes
  report/               Finding types, text/JSON formatters
  schema/               JSON Schema validator (embeds schemas via go:embed)
  semantic/             7 semantic passes: references, uniqueness, statemachines,
                        expressions, sumtypes, surfaces, warnings
schemas/v1/             JSON Schema definition files (also embedded in binary)
  examples/             Reference example + broken test fixtures
  definitions/          14 schema definition files
references/             Language reference, patterns, test generation guide
skills/                 Original skill definitions (validate, distill, elicit)
specs/                  Validator specification
docs/                   Rule and warning documentation
```

## Build and test

```bash
go build -o bin/allium-check ./cmd/allium-check
go test ./...
```

## CLI usage

```bash
bin/allium-check [flags] file1.allium.json [file2.allium.json ...]

Flags:
  --format text|json    Output format (default: text)
  --quiet               Suppress warnings (show errors only)
  --strict              Treat warnings as errors (exit 1)
  --schema-only         Skip semantic checks
  --rules N-M           Only check specific rule numbers
  --version             Print version
```

Exit codes: 0 = clean, 1 = validation errors, 2 = input/parse errors.

## Skills

Three Claude Code skills are available in `.claude/skills/`:

- `/validate` — Run allium-check and present results with explanations
- `/distill` — Extract an Allium spec from existing code
- `/elicit` — Build an Allium spec through guided conversation

## Conventions

- Go module: `github.com/foundry-zero/allium`
- Entity names: PascalCase
- Field names: snake_case
- Inline enum values: snake_case
- Variant names: PascalCase
- 35 validation rules (RULE-01 through RULE-35), 19 warnings (WARN-01 through WARN-19)
