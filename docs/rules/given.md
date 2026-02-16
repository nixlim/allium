# Given Rules

These rules validate `given` bindings — external context provided to the spec at runtime.

---

## RULE-22: Given binding type not declared

A `given` binding references a type (entity or value type) that does not exist in the spec.

**Violation:**
```json
{ "name": "current_user", "type": "UnknownType" }
```
where `UnknownType` is not declared as an entity, external entity, or value type.

**Fix:** Declare the type as an entity, external entity, or value type, or fix the type name.

**Note:** Also documented in [reference.md](reference.md) as this is a reference resolution rule.

---

## RULE-23: Duplicate given binding name

Two `given` bindings share the same name.

**Violation:**
```json
{
  "given": [
    { "name": "current_user", "type": "User" },
    { "name": "current_user", "type": "Admin" }
  ]
}
```

**Fix:** Rename one of the bindings so each has a unique name.

**Note:** Also documented in [uniqueness.md](uniqueness.md) as this is a uniqueness rule.

---

## RULE-24: Given binding requires name and type

Every `given` binding must declare both a `name` and a `type` reference. Enforced by JSON Schema.

**Violation:**
```json
{ "name": "current_user" }
```

**Fix:**
```json
{ "name": "current_user", "type": "User" }
```

**Note:** Also documented in [structural.md](structural.md) as this is a schema-enforced rule.
