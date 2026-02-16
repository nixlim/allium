# State Machine Rules

These rules analyze entity lifecycle state machines built from enum-typed status fields and transition rules.

---

## RULE-07: Unreachable status enum value

A status enum value cannot be reached from any creation point via the transition graph. This indicates a missing rule or an obsolete enum value.

**Violation:** Entity `Order` has status `pending | active | completed | archived` but no rule transitions to `archived`.

**Fix:** Either add a rule that transitions to `archived`, or remove `archived` from the enum if it is no longer needed.

**How it works:** The checker builds a directed graph from creation values (seeds) through all state_change ensures clauses, then runs BFS. Any enum value not visited is unreachable.

---

## RULE-08: Dead-end state with no outgoing transition

A reachable, non-creation status value has no outgoing transitions. This may indicate a missing transition rule or an unintentional terminal state.

**Violation:** Entity `Task` with status `open | blocked | done`. Rules transition `open -> blocked` but nothing transitions from `blocked`.

**Fix:** Add a transition out of `blocked` (e.g., `blocked -> open`) or, if `blocked` is intentionally terminal, suppress the finding.

**Note:** Creation values (seeds) are excluded from this check since they are entry points, not dead ends.

---

## RULE-09: Undeclared status value in assignment

An ensures clause assigns a value to a status field that is not declared in the corresponding enum.

**Violation:** A rule sets `order.status = "cancelled"` but the enum only declares `pending | active | completed`.

**Fix:** Add `cancelled` to the enum, or fix the assigned value to match an existing enum member.

**Scope:** Checks both entity_creation fields and state_change ensures clauses. Validates against both named enumerations and inline enum values.
