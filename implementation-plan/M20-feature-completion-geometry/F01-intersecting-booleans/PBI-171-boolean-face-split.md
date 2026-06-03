---
milestone: M20
feature: F01
pbi: PBI-171
title: Face-splitting solid/solid boolean
status: planned
estimate: XL
---

# PBI-171 — Face-splitting solid/solid boolean

**Milestone:** M20 Feature Completion & Geometry Parity  ·  **Feature:** F01 Intersecting Booleans

## Goal

Extend `ops.Boolean` past Phase-A to handle **intersecting** solids: compute the
surface–surface intersection, imprint split edges on both bodies, classify the
resulting face regions, and assemble Join/Cut/Intersect/NewBody results.

## Scope / work

- Surface–surface intersection → intersection segments (planar faces first:
  segment = the line where two planes cross, clipped to both face boundaries).
- Imprint: split each crossed face along the intersection, inserting new edges and
  vertices with deterministic lineage.
- Region classification using `PointInsideBody` (ray-cast) → inside / outside / on.
- Assemble the kept face set per `PartFeatureOperation`; rebuild a valid manifold body.
- Reference-key lineage on every new face/edge so it rebinds after recompute.

## API contracts (interfaces / enums / collections)

- `ops.Boolean` (extended); no new public DTO (the feature DTOs already exist).

## Acceptance criteria

- Two overlapping axis-aligned boxes: **Join** → one validated manifold solid whose
  volume = V1 + V2 − Voverlap; **Cut** → V1 − Voverlap; **Intersect** → Voverlap.
- Result faces carry stable reference keys that rebind across a recompute.
- Disjoint/containment Phase-A cases still pass (no regression).

## Depends on

M07 (`kernel/ops`, `kernel/topo`, `math/predicate`).

## Notes

The single biggest unblocker in M20 — Cut, Split, Hole, Emboss, and most
sheet-metal/plastic features reduce to this. Start with the axis-aligned/planar
case (exact predicates), generalize to arbitrary planar faces, leave curved–curved
intersection to the NURBS follow-up (F02).
