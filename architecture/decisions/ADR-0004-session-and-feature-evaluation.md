# ADR-0004: Session edits and feature evaluation

Date: 2026-10-10. Status: Accepted for the Rust target design.
Implementation evidence: pending R2-R6.

Boundary: transaction ownership, dependency invalidation, and result adoption.
Owner: session and model modules in `oblikovati-rust`.

## Context

Long recomputes must not stop interaction. A new edit, undo, or document close
can invalidate a result that is still being computed. The Go design supplies
useful dirty-tail and dependency rules, but Rust shared handles require proof
that an evaluation cannot mutate another evaluation's inputs.

## Decision

One session owner commits document edits in order. GUI, CLI, add-ins, and
scripts use the same commands, transactions, undo/redo, and event rules.
Model mutation belongs to commands; UI state and worker results cannot edit
the live document directly. Split session responsibilities into focused modules.

Evaluate an immutable recipe snapshot tagged with document identity and a
monotonic evaluation generation. Replay from the earliest dirty feature using
the last valid clean prefix, through the end-of-part marker. Resolve topology
inputs in that evaluation context using
[ADR-0005](ADR-0005-identity-and-references.md). Suppression and reordering
invalidate the affected dependencies before replay.

Use one typed dependency-key and footprint mechanism for parameters, sketches,
work geometry, and external geometry. An unattributed read causes conservative
recompute; it must not produce silently stale geometry. Full assembly replay
is valid until narrower invalidation is proved. Session orchestration resolves
cross-document dependencies and detects cycles; the model reports its reads
without importing a workspace or filesystem service.

Workers return bodies, provenance, diagnostics, achieved precision, and
presentation updates. The owner adopts a result only for the current document
and generation. New edits, close, undo, and redo invalidate old generations.
Cancel superseded work when possible and discard its result in all cases.
Independent branches may run in parallel only after dependency and shape
isolation tests pass. Start with one evaluation owner per document.

## Interfaces and invariants

- Cloning a Martensite handle does not prove an immutable snapshot. R1 must
  establish deep-copy or isolated-owner behavior. Do not add unsafe `Send` or
  `Sync` implementations to bypass that requirement.
- Commit model results as a coherent generation. The viewport may show the
  last good result while work runs, with its stale/health state visible.
  Exact queries must name the generation they use or await the current one.
- A failed feature reports its faulty entity and quarantines dependents;
  independent work continues. An invalid partial body is diagnostic content,
  never an adopted valid solid. Kernel failures follow
  [ADR-0007](ADR-0007-martensite-integration.md).
- Assembly machining evaluates participant occurrences without mutating the
  shared source part. Component-local references are qualified by occurrence
  identity; selection and persistence must not depend on collection position.
- One completed command produces one undo record and the defined event order.
  Cancelled tools leave no committed preview edits. Recompute itself does not
  append user commands or rewrite stored references.
- Never wait on the session owner from a callback that requires that owner.
  Request timeout semantics remain in [ADR-0002](ADR-0002-api-and-addin-host.md).

## Alternatives and consequences

Mutating the live document on workers would require locks throughout the model
and allow mixed-generation results. Whole-model replay after every edit would
discard the proven clean prefix. Both are rejected. Conservative full replay
remains a correctness measure for unattributed dependencies, with its cost
measured before incremental optimization.

## Legacy disposition

Consolidate Oblikovati ADR-0007, 0010, 0035, and 0044 at the
[fixed revision](legacy-map.md#source-baseline). Retain rollback/replay,
reference inputs, common edit paths, and the never-skip invalidation rule.
Replace goroutine assumptions and closure-based feature inputs with owned
snapshots and serializable definitions. Do not claim that old ADR text proves
asynchronous behavior or dependency coverage in the Go baseline.

## Validation required

Use Go `app/session.go`, `command/`, `event/`, `model/depend/`, feature replay,
and assembly feature tests. R2 tests edit/abort/undo/redo and event order with
named fakes. R3/R6 use real Martensite results for overlapping edits, close
during evaluation, stale-result rejection, parameter footprints, healthy
independent branches, and repeated assembly participants.
