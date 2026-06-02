# Modeling spine (iteration 2)

This iteration layers the **part-modeling spine** onto the iteration-1 core
(`../core/`): the sketch environment + Go-native constraint solver, the feature-
history recompute engine, and the concrete part features — all on Go + the pure-Go
kernel (ADR-0002), driven by the patterns the core established.

It is the realization of the parametric-cad skill's two hardest, highest-value
ideas on this stack: **model = evaluated program** (the feature engine) and
**persistent topological identity** (reference-keyed feature inputs).

## What it covers (plan milestones)

| # | Doc | Modernizes (plan) | New ADRs |
|---|-----|-------------------|----------|
| 00 | [Sketch & constraint solver](00-sketch-and-solver.md) | M06 | [0009](../decisions/ADR-0009-sketch-solver.md) |
| 01 | [Feature-history engine](01-feature-engine.md) | M07-F04, M08-F01 | [0010](../decisions/ADR-0010-feature-recompute-model.md) |
| 02 | [Sketched & work features](02-sketched-work-features.md) | M08-F02/F03/F04 | — |
| 03 | [Dress-up & pattern features](03-dressup-patterns.md) | M09 | — |

## The spine in one diagram

```
   parameters (core/04) ──┐               ┌── reference keys (core/05)
                          ▼               ▼
   sketch entities ─▶ CONSTRAINT SOLVER ─▶ Profile / Path ─┐   (ADR-0009)
   + constraints      (decompose+Newton)                   │
                                                            ▼
                          Definition struct ─▶ features.Add ─▶ FEATURE  (the triangle)
                          (POD + Quantity + Ref)                │
                                                                ▼
   edit ─▶ command (core/06) ─▶ mark dirty (DAG) ─▶ ROLLBACK-REPLAY recompute  (ADR-0010)
                                                     on job pool, immutable snapshot (ADR-0007)
                                                                │
                                                  kernel ops (core/03) ─▶ bodies + lineage
                                                                │
                                          tessellate ─▶ renderer mesh cache (core/08)
                                          Definition struct ─▶ reflection inspector (core/09)
```

## Dependencies on the core (nothing here is new plumbing)

Every mechanism this iteration needs already exists from iteration 1; the modeling
code is almost pure domain logic on top:

- **Dirty-propagation DAG** — same machinery as parameters (core/04) and scene
  transforms (realtime-3d §3); features are just more nodes.
- **Reference keys** (core/05) — feature inputs; the make-or-break seam (ADR-0010).
- **Commands + events** (core/06) — every edit is an undoable command; recompute
  fires before/after events.
- **Async recompute** (ADR-0007) — the engine is pure & cancellable by design.
- **Kernel lineage** (core/03) — the source of reference-key identity.
- **Registry + reflection inspector** (core/07, core/09) — a new feature = a package
  + a `Definition` struct; its ribbon button and edit panel appear for free.

## Kernel-phase reality (ADR-0002)

Feature availability tracks the kernel phase, not this iteration:
- **Phase A (analytic):** extrude, revolve, hole, work features, planar sketch,
  chamfer-on-analytic → a usable analytic part modeler.
- **Phase B (NURBS):** sweep, loft, splines, rolling-ball fillet.
- **Phase C (robust booleans/blends):** combine/split, complex fillets, shell.

The engine, sketch, solver, and the whole UI/command/recompute machinery are built
against Phase A and do **not** wait on the kernel's hard phases — only the *set of
features that succeed* grows as the kernel matures.
