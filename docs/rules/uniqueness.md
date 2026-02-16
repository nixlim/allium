# Uniqueness Rules

These rules ensure that names which must be unique within their scope are not duplicated.

---

## RULE-06: Rules sharing a trigger must have compatible parameters

When multiple rules share the same trigger name, their parameter signatures (count and positional types) must be compatible.

**Violation:** Two rules triggered by `UserSubmitsForm` where one has 2 parameters and the other has 3.

**Fix:** Ensure all rules sharing a trigger name have the same parameter count and compatible types.

---

## RULE-23: Duplicate given binding name

Two `given` bindings declare the same name.

**Violation:**
```json
{
  "given": [
    { "name": "current_user", "type": "User" },
    { "name": "current_user", "type": "Admin" }
  ]
}
```

**Fix:** Rename one of the bindings to be unique.

---

## RULE-26: Duplicate config parameter name

Two `config` parameters declare the same name.

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
