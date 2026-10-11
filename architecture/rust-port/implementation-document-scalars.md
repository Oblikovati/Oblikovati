# Document scalar ownership — partial R3

Tracking: [#3747](https://github.com/Oblikovati/Oblikovati/issues/3747), following
[#3746](https://github.com/Oblikovati/Oblikovati/issues/3746) within
[#3745](https://github.com/Oblikovati/Oblikovati/issues/3745).
The existing [session ownership](../decisions/ADR-0004-session-and-feature-evaluation.md),
[stable identity](../decisions/ADR-0005-identity-and-references.md), and
[parameter](../decisions/ADR-0006-parameters-and-constraints.md) decisions remain
the authority. This extends the unused-entity deletion rule to the stored
[scalar bindings](implementation-scalar-ownership.md), without a second DAG.

## Owner adoption and storage

A scalar binding must resolve to a numeric Model parameter in its document.
Failed formulas remain valid authored inputs, with unavailable values and local
evaluation health. Text parameters converted to Model kind do not satisfy this
numeric requirement. The check runs before history construction, candidate edit
adoption, refresh adoption, typed file registration, and snapshot preparation for
save. Load failure precedes identity registration and GUID allocation. Save
failure leaves the retained package and storage unchanged.

A candidate must retain the original feature and semantic scalar role until it
removes the owner and parameter together. Deleting only the parameter, leaving
an orphaned scalar after owner deletion, or transferring the scalar identity to
another feature is refused. History errors now carry allocator, ownership, and
deletion causes separately; file creation and open retain those typed causes.

## Atomic deletion and consumers

Document datum deletion works on a copy. It removes the feature and all its
owned scalar parameters together, then adopts the complete result. Ordinary
unused feature inputs remain unowned and are retained. Reservation marks remain
outside snapshots and do not rewind through deletion, undo, or redo.

The existing feature/sketch incidence index supplies consumers and unknown read
attribution. Parameter consumers come from stored expression references, even
when cycle handling has detached evaluation graph edges. Consumers outside the
deletion set or unknown reads cause a named refusal. Candidate adoption also
checks removed owners, so direct collection edits cannot bypass this boundary.
Consumers removed in the same edit do not prevent adoption. Unlink a retained
consumer before deleting its source; a compound transaction can group those
ordered edits into one undo step.

Deleting the last hosted sketch prunes its orphaned construction datum through
this same document deletion operation. Its owned scalar is removed too. Existing
construction-chain and ordinary-datum rules remain in force. This is a local
document proof; no new cross-document datum deletion endpoint is exposed.

## Evidence and remaining work

Tests cover missing and wrong-kind IDs, nonnumeric Model values, failed formulas,
history edit and refresh refusal, owner transfer, external scalar consumers,
detached cycle edges, sketch consumers, unknown attribution, construction cleanup,
direct candidate bypass attempts, undo/redo, permanent reservations, load before
registration/entropy, and save before storage change.

Canonical offset creation and redefine still decline pending their authored
expression commands. Their wire contracts already exist. Host group/abort/veto,
source-copy/reopen, unit conversion, and current-health acceptance remain in
[#3745](https://github.com/Oblikovati/Oblikovati/issues/3745). Other feature dimensions will use the same document ownership boundary.
This is Linux validation only and does not certify kernel or viewport completion.
