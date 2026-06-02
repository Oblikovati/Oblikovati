# ADR-0010 — Feature recompute: rollback-replay over reference-keyed inputs

**Status:** accepted · **Context:** the feature-history engine (plan M08-F01,
PBI-087) is the heart of the modeler. Two design commitments need recording: *how*
recompute threads geometry through the history, and *how* features hold topology
inputs across rebuilds.

## Decision

### 1. Recompute is rollback-replay of the dirty tail, on an immutable snapshot

The model is an evaluated program (parametric-cad §0). On any input change we:

1. find the **earliest dirty feature** in history order (via the dependency DAG,
   shared with parameters — core/04);
2. take the cached "current bodies" state from **just before** it (the clean prefix
   is reused, never recomputed);
3. **replay forward** from there, each feature consuming the running body state +
   its resolved inputs and producing new bodies + lineage, up to the end-of-part
   marker;
4. do all of this on the **job pool over an immutable input snapshot** (ADR-0007),
   cancellable, results swapped in at a frame boundary.

This is the classic rollback graph, made **pure and parallel**: independent
branches (separate bodies / unconnected feature chains) replay concurrently; the
main goroutine only adopts results. We explicitly reject whole-model re-evaluation
(too slow) and whole-document snapshots for undo (too expensive — undo is commands,
core/06).

### 2. Feature inputs are reference keys resolved lazily at recompute start

A feature does not hold pointers to the faces/edges/sketches it consumes — those
are destroyed and recreated every rebuild. It holds **`Ref` values** (reference
keys, core/05) that are **resolved at the top of recompute**:

```go
type Ref struct{ key identity.RefKey; ctx *OccurrenceContext } // topology/sketch/workfeature input
type ExtrudeDefinition struct {
    Profile  Ref            // → resolves to a sketch Profile at recompute
    Operation OpKind
    Extent   Extent         // sealed: Distance|ToFace(Ref)|ToNext|ThroughAll|FromTo(Ref,Ref)
    Taper    param.Quantity
}
```

- At recompute, each `Ref` is `Resolve`d against the current topology. **Success →
  proceed. Failure ("reference lost") → the feature goes `Health.Sick`** and
  poisons dependents, never aborting the whole rebuild (parametric-cad §7).
- This is the single wiring that makes edits robust: because keys derive from
  `topo.Lineage` (core/03), a face "the same" after an upstream edit re-resolves;
  a face that genuinely vanished resolves to nothing and the consumer surfaces for
  re-selection.

## Why these two together

They are inseparable: rollback-replay *destroys and recreates* topology every pass,
which is exactly why inputs must be reference keys, not pointers. Committing to both
now — before any concrete feature is written (modeling/02) — is the parametric-cad
skill's central scheduling warning (§7: design identity before features depend on
it). Getting this seam right is what separates a modeler whose edits "just work"
from one that constantly loses references.

## Consequences

- The feature engine API is **pure**: `Recompute(ctx, snapshot) (bodies, lineage,
  health)` — no hidden global state, fully unit-testable headless, trivially the
  body of the gRPC `Edit`/recompute path.
- Every feature's `Definition` is a plain struct whose `Ref` fields *are* the picked
  inputs — so the same struct serializes, drives the reflection inspector (core/09),
  and crosses gRPC, with reference keys as the portable input identity.
- Suppression/conditional-suppression and reorder operate on the history list before
  replay; health propagates along the DAG.

See [modeling/01](../modeling/01-feature-engine.md).
