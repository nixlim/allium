# Sum Type Rules

These rules enforce correct use of entity discriminators (sum types) and variant declarations.

---

## RULE-16: Discriminator variant has no matching variant declaration

An entity's discriminator lists a variant name, but no corresponding `variant X : Entity` declaration exists.

**Violation:** Entity `Node` has discriminator `Branch | Leaf` but only `variant Branch : Node` is declared.

**Fix:** Add the missing variant declaration:
```json
{ "name": "Leaf", "base_entity": "Node", "fields": [] }
```

---

## RULE-17: Variant not listed in base entity discriminator

A variant declaration references a base entity, but the variant name does not appear in that entity's discriminator.

**Violation:** `variant Stem : Node` is declared, but entity `Node` has discriminator `Branch | Leaf` (no `Stem`).

**Fix:** Either add `Stem` to the discriminator or remove the variant declaration.

---

## RULE-18: Variant field accessed without type guard

A rule accesses a field that is specific to a variant without narrowing the type via a type guard (e.g., a requires clause checking the discriminator value).

**Violation:** Accessing `node.children` (a Branch-specific field) without a requires clause like `node.kind = Branch`.

**Fix:** Add a type guard before accessing variant-specific fields:
```
requires: node.kind = Branch
ensures: ... node.children ...
```

**Note:** Type guards include `requires` clauses checking the discriminator and `if` conditions narrowing the type within the ensures block.

---

## RULE-19: Must use variant name for creation when discriminator exists

When an entity has a discriminator, creation must use a specific variant name instead of the base entity name.

**Violation:** A rule creates `Node.created(...)` when `Node` has discriminator `Branch | Leaf`.

**Fix:** Use the variant name: `Branch.created(...)` or `Leaf.created(...)`.
