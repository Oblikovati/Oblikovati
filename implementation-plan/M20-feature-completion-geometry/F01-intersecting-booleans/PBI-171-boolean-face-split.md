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

## Findings 2026-06-05 — robustness gaps (the planar boolean works, but not for every overlap)

The four-stage planar pipeline (imprint → split → classify → stitch) is implemented and
correct for clean overlaps, but a deep modelling session (a parametric fan with lofted blades)
exposed two distinct robustness gaps that produce **non-manifold / inverted-normal** results.
Both are reproduced and isolated; see `kernel/brep/` and the skipped fan e2e in
`oblikovati-mcp-bridge/bridge/e2e_fan_validate_test.go` (the acceptance spec for the fix).

1. **Partial penetration is not cut (the core gap).** `splitFace` runs the 2D arrangement
   (`Arrange`/`traceCycles`) which only subdivides a face when the imprint segments form a
   **closed loop** on it. When a tool pokes **part-way** into a face — or crosses a **concave
   faceted wall** (e.g. a blade spanning out through a 32-gon bore) — the per-face-pair imprint
   produces **dangling/partial segments** whose loop closes only *across* adjacent face
   boundaries. `traceCycles` drops the dangling edges → the face is never split → the
   overlapping material is not removed → the tool stitches as a coincident shell (edges used by
   3–4 faces, or a manifold-but-mis-oriented result). Instrumented evidence: a fan blade join
   had `imprintSegs=86` but only ~13 face subdivisions. **The fix must assemble imprint segments
   into closed loops across face boundaries and handle T-vertices / dangling edges in the
   arrangement** — see PBI-199.

2. **Coincident / near-tangent faces** (the documented "coplanar follow-up"). `boolean_coplanar.go`
   handles area-coincident coplanar faces, but a tool face that is *coincident with* or *grazes
   near-tangent to* the target (especially a faceted curved wall) is not robustly resolved.

Dead-end recorded so it is **not retried**: a post-stitch *orientation-repair* pass (flood-fill
orientation across shared edges, flip the minority) does NOT fix the inverted normals — the
mis-orientation is a symptom of the un-cut overlap (1), not a cleanly re-orientable manifold, so
flood-fill cannot resolve it. The fix must act at the imprint/arrangement stage, not on the output.

Interim safety net (unchanged): `ops.booleanGeneral` already falls back to triangle-soup BSP CSG
when `brep.Boolean` returns `ErrNonPlanar`; a guarded "CSG-repairs-a-non-manifold-planar-result"
fallback was prototyped but rejected (turns clean B-reps into heavy triangle soup and still
degrades under pattern chaining).

### Related fix that landed (precondition for this boolean)

Twisted **lofts/sweeps/revolves** were emitting **warped (non-planar) ruled quad** side faces as
single "planar" faces, so the boolean imprinted against an *approximating* plane and segments
landed offset from the true edges (~0.875·twist; even 0.001 rad broke the loop closure). FIXED in
`model/feature/swept.go` (`sideQuad`/`quadPlanar` triangulate a non-coplanar side quad) — see
PBI-174. This restores the planar-faceted invariant the boolean requires; it is a separate fix
from the arrangement gap above.
