# Structural Rules (Schema-Enforced)

These rules are enforced by JSON Schema validation. They catch malformed documents before semantic analysis runs.

---

## RULE-02: Every field must declare a type

Every entity field must include a `type` property with a valid `FieldType` discriminator.

**Violation:**
```json
{ "name": "email" }
```

**Fix:**
```json
{ "name": "email", "type": { "kind": "primitive", "value": "String" } }
```

---

## RULE-04: Every rule must have a trigger and non-empty ensures

Rules require both a `trigger` object and an `ensures` array with at least one clause.

**Violation:**
```json
{ "name": "DoNothing", "trigger": { "kind": "external_stimulus", "name": "noop" }, "ensures": [] }
```

**Fix:** Add at least one ensures clause, or remove the rule if it has no effects.

---

## RULE-05: Trigger kind must be one of 7 valid kinds

Valid trigger kinds: `external_stimulus`, `state_transition`, `temporal`, `observation`, `lifecycle`, `system_event`, `derived_trigger`.

**Violation:**
```json
{ "kind": "custom_trigger", "name": "my_trigger" }
```

**Fix:** Use one of the 7 recognized trigger kinds.

---

## RULE-15: Discriminator variant names must be PascalCase

Variant names in entity discriminators must follow PascalCase naming (e.g., `Branch`, `Leaf`).

**Violation:**
```json
{ "discriminator": ["branch", "leaf"] }
```

**Fix:**
```json
{ "discriminator": ["Branch", "Leaf"] }
```

---

## RULE-20: Enumeration values must be non-empty

Named enumerations must declare at least one value.

**Violation:**
```json
{ "name": "Status", "values": [] }
```

**Fix:** Add at least one value to the enumeration.

---

## RULE-21: Variant declaration requires name and base_entity

Every variant declaration must specify both a `name` and the `base_entity` it belongs to.

**Violation:**
```json
{ "name": "Branch" }
```

**Fix:**
```json
{ "name": "Branch", "base_entity": "Node" }
```

---

## RULE-24: Given binding requires name and type

Each `given` binding must declare a name and a type reference.

**Violation:**
```json
{ "name": "current_user" }
```

**Fix:**
```json
{ "name": "current_user", "type": "User" }
```

---

## RULE-25: Config parameter requires name, type, and default_value

Every config parameter must specify name, type, and a default value.

**Violation:**
```json
{ "name": "max_retries", "type": { "kind": "primitive", "value": "Integer" } }
```

**Fix:**
```json
{
  "name": "max_retries",
  "type": { "kind": "primitive", "value": "Integer" },
  "default_value": { "kind": "literal", "type": "Integer", "value": 3 }
}
```
