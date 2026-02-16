# Allium Validation Warnings

Warnings indicate potential issues that do not prevent the spec from being valid. Exit code remains 0 unless `--strict` is used.

---

## WARN-01: External entity has no governing spec

An external entity is declared but not associated with any `use_declaration` import.

**Trigger:** `external_entities` contains an entity not covered by any `use_declarations`.

**Resolution:** Add a `use_declaration` with a coordinate pointing to the governing spec, or document why the entity is unlinked.

---

## WARN-02: Open questions present

The spec contains unresolved open questions.

**Trigger:** The `open_questions` array is non-empty.

**Resolution:** Resolve each open question and remove it from the array, or acknowledge them as intentionally deferred.

---

## WARN-03: Deferred spec has no location hint

A deferred specification entry has a null or empty `location_hint`.

**Trigger:** `deferred` entry with `"location_hint": null`.

**Resolution:** Add a location hint indicating where the deferred spec will eventually be defined.

---

## WARN-04: Unused entity or field

An entity or field is declared but never referenced by any rule, surface, relationship, or other entity.

**Trigger:** Entity `Archive` exists but no rule, surface, or relationship references it.

**Resolution:** Remove the unused declaration or add rules/surfaces that reference it.

---

## WARN-05: Rule can never fire (contradictory requires)

A rule's requires clauses are mutually exclusive, making the rule impossible to trigger.

**Trigger:** `requires: status = "active" and status = "pending"` (status cannot be both).

**Resolution:** Fix the contradictory conditions or remove the rule.

---

## WARN-06: Temporal rule has no re-firing guard

A temporal trigger has no requires clause to prevent it from firing repeatedly on the same entity.

**Trigger:** A temporal rule that checks `expires_at < now` without guarding against re-processing.

**Resolution:** Add a requires clause such as `status != "expired"` to prevent re-firing.

---

## WARN-07: Surface exposes unused field

A surface exposes a field that is not used by any rule in the system.

**Trigger:** Surface exposes `order.archived_at` but no rule reads or writes `archived_at`.

**Resolution:** Remove the unused exposure or add rules that use the field.

---

## WARN-08: Provides has impossible when condition

A surface provides clause has a `when` condition that can never be true.

**Trigger:** `when: status = "active" and status = "pending"`.

**Resolution:** Fix the condition or remove the provides clause.

---

## WARN-09: Unused actor

An actor is declared but never referenced in any surface `facing` clause.

**Trigger:** Actor `AdminUser` is declared but no surface faces `AdminUser`.

**Resolution:** Add a surface facing the actor or remove the unused declaration.

---

## WARN-10: Sibling rule creates entity without duplicate guard

A rule creates a child entity for a parent without checking whether a duplicate already exists.

**Trigger:** Rule creates `LineItem` for `Order` without requires-guarding against existing items.

**Resolution:** Add a requires clause that prevents duplicate creation (e.g., checking item count or existence).

---

## WARN-11: Provides condition weaker than rule requires

A surface provides a trigger with a `when` condition that is strictly weaker than the corresponding rule's `requires` clause, meaning the action may be presented when the rule cannot actually fire.

**Trigger:** Surface shows "Activate" button when `status != "done"`, but the rule requires `status = "pending"`.

**Resolution:** Tighten the provides `when` condition to match or be stricter than the rule's requires.

---

## WARN-12: Overlapping preconditions on shared trigger

Two rules sharing the same trigger have requires clauses that could both be true simultaneously, creating ambiguity about which rule fires.

**Trigger:** Rules `ApproveSmall` and `ApproveLarge` both trigger on `approve_order`. `ApproveSmall` requires `amount < 1000`, `ApproveLarge` requires `amount > 500` — amounts 500-1000 match both.

**Resolution:** Make the requires clauses mutually exclusive or document the intended overlap.

---

## WARN-13: Derived value references out-of-entity field

A parameterised derived value references a field that belongs to a different entity.

**Trigger:** Entity `Order` has derived value `total` that references `customer.discount_rate`.

**Resolution:** Pass the external value as a parameter or restructure the derivation.

---

## WARN-14: Trivial actor identified_by condition

An actor's `identified_by` condition always evaluates to true or always to false.

**Trigger:** `identified_by: true` or `identified_by: 1 = 2`.

**Resolution:** Provide a meaningful condition that distinguishes the actor.

---

## WARN-15: All-conditional ensures with empty path

All ensures clauses in a rule are inside conditionals, and at least one branch produces no effects.

**Trigger:** A rule where every ensures clause is inside `if/else` blocks, with an else branch that has no clauses.

**Resolution:** Add ensures clauses for the empty branch or document why the no-op path is intentional.

---

## WARN-16: Temporal trigger on optional field

A temporal trigger references an optional field (`T?`) which may be absent, preventing the trigger from ever firing.

**Trigger:** Temporal trigger on `subscription.expires_at` where `expires_at` is typed as `Timestamp?`.

**Resolution:** Ensure the field is always set before the temporal trigger is expected to fire, or use a non-optional type.

---

## WARN-17: Raw entity type used when actors available

A surface uses a raw entity type in its `facing` clause when actor declarations exist for that entity.

**Trigger:** Surface faces `User` directly when `AdminUser` and `RegularUser` actors are defined for `User`.

**Resolution:** Use the specific actor type in the facing clause for proper access control.

---

## WARN-18: transitions_to fires on creation value

A `transitions_to` trigger fires on a status value that entities can be created with, meaning the trigger may fire during creation rather than only on transitions.

**Trigger:** `transitions_to: status = "active"` where creation also sets `status = "active"`.

**Resolution:** Guard the rule to distinguish creation from transition, or use a different trigger kind.

---

## WARN-19: Multiple identical inline enums suggest named enum

The same entity has multiple fields with identical inline enum literal sets, suggesting a named enumeration would be clearer.

**Trigger:** Entity has `priority: "low" | "medium" | "high"` and `severity: "low" | "medium" | "high"`.

**Resolution:** Extract a named enumeration (e.g., `Level`) and reference it from both fields.
