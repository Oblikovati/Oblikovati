---
milestone: M25
feature: F03
pbi: PBI-330
title: Watertight curved-face tessellation (shared edge discretization everywhere)
status: planned
estimate: L
---

# PBI-330 — Watertight curved-face tessellation

**Milestone:** M25 Imported B-Rep Healing  ·  **Feature:** F03 Face sewing into watertight shells

## Why (measured root cause, 2026-06-08)

The tessellated body is **not watertight** on bodies with curved analytic faces — EDF body3 (the duct)
has ~54–69 free (unpaired) edges in BOTH the baseline (`trimmedPatchMesh`) and the metric mesher
(`nurbsPcurveMesh`). The non-watertightness is **pre-existing**, not caused by the metric mesher
(baseline 54, metric 69 — same order).

The design intent (`kernel/ops/edge_discretize.go`) is that every face samples a shared edge into the
SAME chord polyline via `discretizeEdge`, so neighbours meet crack-free. Planar (`earcut`) and B-spline
(`trimmedPatchMesh`/`nurbsPcurveMesh`) faces obey this. But the analytic grid meshers do NOT:

- `structuredGridMesh` (cylinder/cone walls) tessellates over its own `(u,v)` grid (`isoRectangleGrid`),
  sampling each boundary iso-curve at the grid lines — not at the shared edge's `discretizeEdge` points.
- `gridPatchMesh` (sphere caps) likewise meshes over its own `(u,v)` sampling.

So at any edge between a grid-meshed analytic face and a neighbour, the two samplings differ (interleaved,
neither a subset) → the edge's two triangle fans don't share vertices → free edges.

## Impact

A non-watertight mesh has no well-defined divergence volume, so **per-body mass-properties are wrong**
(the EDF +33% — the leaks corrupt the sum; the baseline total ≈ OCC is luck-of-cancellation). It also
**blocks the curved-boolean CSG fallback and STL/3MF export on imported geometry** (both consume this
tessellation via the same `TessellateBody`). So watertightness is a prerequisite for operating on
imported NURBS bodies, not just a cosmetic/volume nicety.

## Approach (preferred: conform the grid meshers to the shared edge)

Make `structuredGridMesh` and `gridPatchMesh` use the shared `discretizeEdge` polyline on their
boundary iso-curves, so all faces meet on identical edge points:

- For each boundary iso-curve of the grid, replace the grid-line samples with the shared edge's
  `discretizeEdge` points (mapped to `(u,v)` via `ParamAt`), and triangulate the transition between the
  (possibly denser) boundary samples and the interior grid rows without T-junctions (a graded fan, or
  fall back to the CDT for the boundary band).
- Acceptance: every interior edge of an assembled body is shared by exactly 2 triangles (0 free edges)
  on EDF + the OCC fixtures.

### Alternative (mesher-agnostic: a mesh sew post-process)

A `TessellateBody` post-process that welds coincident vertices, then **zips** remaining free edges by
merging the two neighbours' edge samplings into a common refinement (insert each face's points onto the
other's edge — T-junction removal). More general but heavier; the conform approach is preferred because
it keeps each face's mesh correct by construction.

## Acceptance criteria

- EDF.STEP: **0 free edges per body** (watertight), verified by a committed manifold check on the
  assembled body mesh.
- EDF total volume within tolerance of OCC `getMass` (~207,002) with the metric mesher — the leaks were
  the +33%; closing them should land it.
- A curved boolean (e.g. cut a cylinder from an imported NURBS body) succeeds and is volume-correct.
- The OCC oracle + full kernel suite stay green; lint clean.
- Live: EDF still renders smooth (no regression) and STL export of EDF is watertight.

## Depends on

`kernel/ops/{tessellate_trim,structured_grid,refined_patch,edge_discretize}.go`, `discretizeEdge`
(the canonical shared sampling), the OCC oracle. Closely related to F03 sewing (this is sewing at the
tessellation level).
