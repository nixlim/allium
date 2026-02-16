# Config Rules

These rules validate `config` parameters — system-level configuration with required defaults.

---

## RULE-25: Config parameter requires name, type, and default_value

Every config parameter must specify all three fields. Enforced by JSON Schema.

**Violation:**
```json
{ "name": "max_retries", "type": { "kind": "primitive", "value": "Integer" } }
```
(missing `default_value`)

**Fix:**
```json
{
  "name": "max_retries",
  "type": { "kind": "primitive", "value": "Integer" },
  "default_value": { "kind": "literal", "type": "Integer", "value": 3 }
}
```

**Note:** Also documented in [structural.md](structural.md) as this is a schema-enforced rule.

---

## RULE-26: Duplicate config parameter name

Two config parameters share the same name.

**Violation:**
```json
{
  "config": [
    { "name": "max_retries", "type": { "kind": "primitive", "value": "Integer" }, "default_value": 3 },
    { "name": "max_retries", "type": { "kind": "primitive", "value": "Integer" }, "default_value": 5 }
  ]
}
```

**Fix:** Remove the duplicate or rename one parameter.

**Note:** Also documented in [uniqueness.md](uniqueness.md) as this is a uniqueness rule.

---

## RULE-27: Config parameter referenced but not declared

An expression references a config parameter name that does not appear in the `config` array.

**Violation:**
```json
{ "kind": "field_access", "field": "missing_param", "object": { "kind": "field_access", "field": "config" } }
```

**Fix:** Declare the config parameter or fix the reference name to match an existing parameter.

**Note:** Also documented in [reference.md](reference.md) as this is a reference resolution rule.
