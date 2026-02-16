# Surface Rules

These rules validate surfaces (boundary contracts) ensuring field paths are reachable, bindings are used, and conditions reference valid fields.

---

## RULE-29: Unreachable path in surface exposes

An `exposes` entry references a field path that is not reachable from the surface's `facing`, `context`, or `let` bindings.

**Violation:** Surface with `facing: viewer: User` and `context: order: Order` exposes `product.name`, but `product` is not reachable from `viewer` or `order`.

**Fix:** Ensure the exposed path starts from a facing, context, or let binding. Add a let binding if the path requires intermediate navigation.

---

## RULE-32: Unused binding in surface

A `facing` or `context` binding is declared but never referenced in the surface body (exposes, provides, related, or let bindings).

**Violation:** Surface declares `facing: viewer: User` but `viewer` is never used anywhere in the surface.

**Fix:** Either use the binding in the surface body or remove it if unnecessary.

---

## RULE-33: Invalid when condition reference in surface

A `when` condition in a provides or exposes clause references a field that is not reachable from the surface's facing or context bindings.

**Violation:** A provides clause has `when: unknown.field` where `unknown` is not a facing, context, or let binding.

**Fix:** Ensure when conditions only reference fields reachable from declared bindings.

---

## RULE-34: Cannot iterate over non-collection type

A `for_each` provides clause targets a field that is not a collection type (e.g., iterating over a String or Integer).

**Violation:** `for_each` over a field typed as `String`.

**Fix:** Ensure the collection expression resolves to a list, set, or other collection type.
