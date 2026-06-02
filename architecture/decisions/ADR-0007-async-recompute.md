# ADR-0007 — Decouple recompute from the frame loop (async, progressive)

**Status:** accepted · **Context:** parametric recompute is heavy; the view loop
must stay at 60fps. (realtime-3d §2, §13; parametric-cad skill §2)

## Decision

The **frame loop never blocks on recompute.** Feature-history recompute and
tessellation run on the **job pool** (worker goroutines). The render loop always
draws the **last good tessellation**; when a recompute job finishes, its results
are swapped in at a frame-phase boundary (deferred runner).

## Why this is a modernization, not just a port

Inventor (and most COM-era MCAD) recompute **synchronously** — the UI freezes
during a long rebuild. With Go's concurrency we can do better:

- **Responsive view during edits.** Orbit/pan/zoom and even start another edit
  while a rebuild proceeds in the background.
- **Progressive results.** Independent feature branches and independent bodies
  recompute in parallel (the dependency DAG, parametric-cad §2, parallelized like
  the transform DAG in realtime-3d §3); finished bodies appear as they complete.
- **Cancellation.** A newer edit supersedes an in-flight recompute via
  `context.Context` cancellation; stale jobs drop their results.

## The model

```
edit → command applied → mark dirty features (DAG) → enqueue recompute job(ctx)
                                                          │ (worker pool)
   frame loop keeps rendering last-good meshes  ◀─────────┤
                                                          ▼
   job computes dirty tail in dependency order → tessellate changed bodies
                                                          │
   deferred runner (frame boundary): swap GPU meshes, clear dirty, fire "recomputed"
```

Invariants:
- **Document mutation happens only on the main goroutine**, inside commands, at
  phase boundaries — workers compute on an immutable snapshot/inputs and return
  results; they never mutate the live document. This avoids locks on the model.
- **Recompute is pure** w.r.t. its inputs (parameters + reference-key-resolved
  topology), so it is safely parallelizable and cancellable.
- **One in-flight recompute per document**, superseding; finer-grained pipelining
  is a later optimization.

## Costs / mitigations

- **Showing stale geometry briefly** → a subtle "recomputing…" affordance; edits
  that *must* be synchronous (rare, e.g. measuring exact current state) can await
  the job.
- **Snapshot cost** → recompute reads parameters by value and topology by reference
  key; the snapshot is cheap (inputs, not the whole B-rep).

## Consequences

The feature engine (iteration 2, M07–M08) is designed **pure and cancellable** from
the start — inputs in, bodies out, no hidden global state — which also makes it
unit-testable headless. The frame loop's phases (core/00) include a
`recompute-sync` phase that only swaps completed results.
