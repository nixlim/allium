# Expression Rules

These rules validate expressions in derived values, requires clauses, filters, and ensures clauses.

---

## RULE-10: Cycle detected in derived value dependencies

Derived values form a dependency cycle. A depends on B and B depends on A (directly or transitively), creating an infinite loop.

**Violation:** Entity `Order` has derived value `total` referencing `tax`, and `tax` referencing `total`.

**Fix:** Break the cycle by computing one value independently or restructuring the derivation.

**How it works:** The checker builds a dependency graph of all derived values and runs Tarjan's SCC algorithm. Any strongly connected component with more than one member (or a self-loop) is a cycle.

---

## RULE-11: Identifier not in scope

A field access path references a root identifier that is not available in the current scope.

Valid scope sources: trigger bindings, for-clause bindings, let bindings, given bindings, config parameters, and defaults.

**Violation:** A rule's requires clause accesses `unknown_binding.status` where `unknown_binding` is not defined.

**Fix:** Ensure the root identifier matches a trigger binding, `given` name, `config` reference, or other in-scope binding.

---

## RULE-12: Type mismatch in expression

An expression uses incompatible types in a comparison or arithmetic operation.

**Violation examples:**
- Comparing Integer to String: `order.amount = "hello"`
- Arithmetic on Boolean: `flag + 1`
- Comparing Timestamp to Integer: `order.created_at < 42`

**Valid special cases:**
- `Timestamp - Duration` produces a Timestamp (date arithmetic)
- `Timestamp - Timestamp` produces a Duration

**Fix:** Ensure both sides of comparisons share compatible types, and arithmetic operates on numeric or temporal types.

---

## RULE-13: Collection operation missing explicit lambda parameter

An `any`, `all`, or similar collection operation does not declare an explicit `lambda_param` for the iteration variable.

**Violation:**
```json
{ "kind": "any", "collection": { "kind": "field_access", "field": "items" }, "predicate": { ... } }
```
(missing `lambda_param`)

**Fix:** Add an explicit `lambda_param`:
```json
{ "kind": "any", "collection": { ... }, "lambda_param": "item", "predicate": { ... } }
```

---

## RULE-14: Cannot compare inline enums from different fields

Inline enum fields from different declarations cannot be compared because they have no shared type identity. Named enums of the same type can be compared.

**Violation:** Comparing `order.status` (inline enum `pending | active`) with `task.state` (inline enum `open | closed`).

**Fix:** Either use named enumerations for both fields (giving them a shared type) or restructure the comparison.

**Note:** Comparing a field against a literal value of the same inline enum is valid. Only cross-field comparisons between different inline enums are rejected.
