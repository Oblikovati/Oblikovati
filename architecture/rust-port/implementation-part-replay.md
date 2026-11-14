# Ordered datum and prism replay — partial R1/R3

The model now executes stored datum and selected-prism inputs through one
`FeatureReplayController`. This implements part of
[ADR-0004](../decisions/ADR-0004-session-and-feature-evaluation.md) and the
model-owned interface in [ADR-0007](../decisions/ADR-0007-martensite-integration.md).
There is one ordered health report for every active feature. Each replay starts
with empty derived result tables; authored recipes remain unchanged.

Datum entries use the existing datum implementation. Prism entries resolve their
current scalar inputs, fully constrained sketch, host frame, orientation, and
projected point sources before calling the injected `SketchPrismEvaluator`.
Placement and projections borrow the current datum catalogue from this replay.
The backend receives checked inputs and does not read or replay the document.
`MartensitePrismEvaluator` uses the existing selected-profile construction,
validation, provenance checks, and isolated diagnostic snapshot path.

Failures retain their source kind: datum, metadata, input preparation, or kernel adapter.
Failed features block their immediate readers. Independent entries continue.
Blocked dependencies take precedence over suppression, as required by ADR-0004.
Suppressed, cutoff, and failed entries have no current body output. A fresh replay
can recover without changing earlier reports or serving last-good bodies.

`retain_stored_sketch_prism` now consumes this common replay. Its separate
suppression and active-entry helpers were deleted. No geometry algorithm,
recognizer, tolerance constant, or fallback was added. Kernel construction and
meshing remain in Martensite.

Tests use a named fake backend to check calls and local failure. Released-kernel
tests check a directed-boundary decline beside a valid independent prism, and a
failed host plane followed by recovery. Existing owner edit, undo, package,
condition, placed-frame, and native pixel tests consume the same path.

This does not close the complete extrude contract. Prefix reuse, cancellation,
generation-scoped session adoption, accumulated body state for later operations,
full validity levels, achieved tolerance budgets, persistent naming, and general
scene publication remain open. The retained prism is still a diagnostic kernel
result, not a claim that all operation-result requirements are met.

Run `CARGO_NET_OFFLINE=true make gate` for the complete local Linux checks.

A regression first proved that a raw snapshot with a prism construction-datum
flag called the backend. Replay now validates prism metadata before resolving
inputs or constructing geometry and records a typed metadata failure. The test
checks that the backend receives no call and no body output is available.
