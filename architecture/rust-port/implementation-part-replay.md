# Ordered datum and prism replay — partial R1/R3

The model now executes stored datum and selected-prism inputs through one
`FeatureReplayController`. This implements part of
[ADR-0004](../decisions/ADR-0004-session-and-feature-evaluation.md) and the
model-owned interface in [ADR-0007](../decisions/ADR-0007-martensite-integration.md).
There is one ordered health report for every active feature. A full replay starts
with empty derived result tables; a guarded tail replay starts at a retained
feature boundary. Authored recipes remain unchanged.

Datum entries use the existing datum implementation. Prism entries resolve their
current scalar inputs, fully constrained sketch, host frame, orientation, and
projected point sources before calling the injected `PartFeatureEvaluator`.
Fillet and Chamfer use its explicit blend operation; a partial evaluator must
return a named decline before it constructs a blend. (#4079)
Named planar-face resolution also requires an explicit evaluator implementation.
Placement and projections borrow the current datum catalogue from this replay.
The backend receives checked inputs and does not read or replay the document.
`CertifiedNamedSolidEvaluator` uses selected-profile construction, validation,
and provenance checks for both diagnostic and certified replay.

Failures retain their source kind: datum, metadata, input preparation, or kernel adapter.
Failed features block their immediate readers. Independent entries continue.
Blocked dependencies take precedence over suppression, as required by ADR-0004.
Suppressed, cutoff, and failed entries have no current body output. A fresh replay
can recover without changing earlier reports or serving last-good bodies.

`retain_stored_sketch_prism` takes a document GUID and consumes this complete
named replay before returning a detached diagnostic prism. Its separate
suppression and active-entry helpers were deleted. No geometry algorithm,
recognizer, tolerance constant, or fallback was added. Kernel construction and
meshing remain in Martensite.

Tests use a named fake backend to check calls and local failure. Released-kernel
tests check a directed-boundary decline beside a valid independent prism, and a
failed host plane followed by recovery. Existing owner edit, undo, package,
condition, placed-frame, and native pixel tests consume the same path.

The model stores each captured candidate's source generation and replay inputs.
The session guards checkpoint capture and adoption by document, generation, and
accepted edit. Each captured named body also checks that its own source history
matches the accepted edit, including when document identifiers are equal. The
adapter still depends on the session for this guarded completion; moving the
result store into the session must preserve that boundary. The replay owner
reuses checkpoints for blend creation and edits
when policy and sketch inputs permit it. Associative sketches use ordered
recompute. General dirty-prefix replay, session ownership, full validity levels,
achieved tolerance budgets, and operation-result requirements remain open.

Run `CARGO_NET_OFFLINE=true make gate` for the complete local Linux checks.

A regression first proved that a raw snapshot with a prism construction-datum
flag called the backend. Replay now validates prism metadata before resolving
inputs or constructing geometry and records a typed metadata failure. The test
checks that the backend receives no call and no body output is available.
