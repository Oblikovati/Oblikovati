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

The model seals each captured result to its evaluated inputs, source generation,
and accepted edit. The document session owns the result and its checkpoints,
records the policy, and checks each transfer before adoption. Each captured
named body also checks its own source history, even when document IDs match.
The adapter completes transfers through the model without a session dependency.
Prefixes share immutable results and retire when their slot is replaced or the
document closes. Certified Extrude creation can adopt its validated prefix replay
directly; a validator without this capability keeps its own validation path.
Extrude previews borrow a current prefix and discard the private candidate on
cancel. A different policy or an associative sketch uses full evaluation. Parameter changes use current feature and sketch reads to find
the earliest dirty boundary; unclassified inputs restart replay. Martensite
checkpoint tests compare the changed Fillet and each named face with a full
replay. Parameter Set and Update use a current checkpoint when its clean
prefix and independent sketches allow it. First-feature changes, associative
sketches, and open transactions retain the full-replay path. General dirty-prefix replay, full
validity levels, achieved tolerance budgets, and operation-result requirements
remain open.

Sketch inspection now checks the requested sketch's placement and point
projections before asking the document owner for named datums. An independent
sketch Get starts zero named replays even when another sketch in the same Part
uses a face. A Get for the face sketch still fetches current owner datums.
List requests check the dependencies of every sketch they return.
See the [sketch inspection replay receipt](receipts/2026-11-24-sketch-inspection-replay-4079/README.md).
The desktop's dimension count uses that same dependency check and reuses
current datums under the document's policy. Cold read-only inspection uses the
session's generation-sealed replay request and leaves the transaction and
published results intact. See the
[native sketch support cases](receipts/2026-11-24-owned-sketch-datums-4079/README.md).

Feature Properties uses the same checkpoint edit path as other feature
edits. Applying conditional suppression to a later feature can publish
the new ordered result during the edit when the document has a current
checkpoint and replay proves the earlier inputs are unchanged.
See the [Feature Properties receipt](receipts/2026-11-24-feature-properties-checkpoint-4079/README.md).
The typed suppression command can also read the changed feature's health from
its candidate tail and publish that replay with the accepted edit. It uses this
path only for a current checkpoint under the certified backend's policy.
The backend still validates edits that require full replay. See the
[typed suppression receipt](receipts/2026-11-24-typed-suppression-checkpoint-4079/README.md).

Typed feature renames use the same checkpoint edit path as feature reorder.
Renaming the later Fillet in a Part can preserve the certified Extrude prefix
and its face references. See the
[typed rename receipt](receipts/2026-11-24-typed-rename-checkpoint-4079/README.md).

Typed Delete Body replays the appended deletion against the retained source
bodies. It publishes the updated checkpoints before listing the surviving
bodies, including when the result is empty. See the
[body deletion receipt](receipts/2026-11-24-body-delete-checkpoint-4079/README.md).

Run `CARGO_NET_OFFLINE=true make gate` for the complete local Linux checks.

A regression first proved that a raw snapshot with a prism construction-datum
flag called the backend. Replay now validates prism metadata before resolving
inputs or constructing geometry and records a typed metadata failure. The test
checks that the backend receives no call and no body output is available.
