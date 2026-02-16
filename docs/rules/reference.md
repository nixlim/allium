# Reference Resolution Rules

These rules ensure every name reference in a spec resolves to a declared symbol. Reference errors are the most common class of spec errors.

---

## RULE-01: Entity referenced but not declared

An `entity_ref` type references an entity name that does not appear in `entities`, `external_entities`, or `use_declarations`.

**Violation:**
```json
{ "kind": "entity_ref", "entity": "FooBar" }
```
where `FooBar` is not declared anywhere.

**Fix:** Declare the entity, add it as an external entity, or fix the typo.

---

## RULE-03: Relationship target entity not declared

A relationship's `target_entity` does not match any declared entity.

**Violation:**
```json
{ "name": "owner", "target_entity": "NonExistent", "kind": "belongs_to" }
```

**Fix:** Ensure the target entity is declared in `entities` or `external_entities`.

---

## RULE-22: Given binding type not declared

A `given` binding references a type (entity or value type) that is not declared.

**Violation:**
```json
{ "name": "current_user", "type": "UnknownType" }
```

**Fix:** Declare the type as an entity, external entity, or value type.

---

## RULE-27: Config parameter referenced but not declared

An expression references a config parameter name that does not appear in the `config` array.

**Violation:**
```json
{ "kind": "field_access", "field": "missing_param", "object": { "kind": "field_access", "field": "config" } }
```

**Fix:** Declare the config parameter or fix the reference name.

---

## RULE-28: Surface facing type not declared

A surface's `facing` clause references a type that does not match any declared entity or actor.

**Violation:**
```json
{ "facing": { "binding": "viewer", "type": "UnknownActor" } }
```

**Fix:** Declare the actor or entity, or fix the type name.

---

## RULE-30: Surface provides trigger not declared

A surface `provides` clause references a trigger name that does not match any declared rule's trigger.

**Violation:**
```json
{ "kind": "action", "trigger": "NonExistentRule", "arguments": [] }
```

**Fix:** Ensure a rule exists with a matching trigger name.

---

## RULE-31: Surface related surface name not declared

A surface `related` clause references a surface name that is not declared.

**Violation:**
```json
{ "surface": "MissingSurface", "context_expression": { "kind": "field_access", "field": "id" } }
```

**Fix:** Declare the related surface or fix the name.

---

## RULE-35: Use declaration imports unresolvable type

A `use_declaration` imports a type that cannot be resolved from the referenced external specification.

**Violation:**
```json
{ "coordinate": "abc123", "alias": "ext", "imports": ["ExternalType"] }
```
where `ExternalType` does not exist in the external spec.

**Fix:** Verify the imported type name matches what the external spec exports.
