---
milestone: M24
feature: F02
pbi: PBI-321
title: 3D-space interior refinement (replaces the (u,v) grid)
status: planned
estimate: L
---

# PBI-321 — 3D-space interior refinement (replaces the (u,v) grid)

**Milestone:** M24 Tolerant NURBS Surface Meshing  ·  **Feature:** F02 Trim-respecting adaptive interior

## Why (supersedes PBI-316's interior)

PBI-316 proved a `(u,v)`-grid interior cannot refine imported NURBS faces: their parameterization is
**non-conformal**, so a `(u,v)` point inside the trim maps ~15% of the face size OFF the trim
surface (over-encloses, +33% volume). The interior must be refined in **3D space**, where new points
stay near the true surface and `ParamAt` is reliable.

## Approach

Refine the **boundary-only** mesh (`trimmedPatchMesh` — watertight, correct volume, but it folds /
is coarse on a curved face) by **3D subdivision + surface projection**:

- Repeatedly take an interior triangle that is too coarse (its `ParamAt→PointAt` centroid deviates
  from the triangle plane by more than `q.ChordTolerance`) or that folds, insert the **projected
  centroid** (`PointAt(ParamAt(centroid))` — the centroid lies on the chord, near the surface, so
  the projection is reliable, unlike a `(u,v)` grid point), and re-triangulate locally.
- **Never subdivide a boundary edge** — boundary vertices stay the exact shared edge points, so the
  body stays watertight (no T-junctions with neighbour faces). Only interior edges/faces refine.
- Run PBI-315 fold repair after; the new points follow the real surface, so the chord-fold the
  boundary-only mesh had is removed.
- The refinement is curvature-adaptive (PBI-314's deflection idea, but driven by 3D triangle
  deviation, not a `(u,v)` step), so flat regions stay coarse.

This keeps the watertight boundary + correct volume of `trimmedPatchMesh` and adds curvature in 3D,
sidestepping the non-conformal `(u,v)` entirely.

## Acceptance criteria

- **EDF.STEP**: external freeform faces fold-free (committed fold detector → 0) AND total volume
  within tolerance of OCC `getMass` (no inflation; the boundary-only baseline is 210k / OCC 207k).
- A conformal dome patch refines (more triangles than boundary-only) AND a constructed strongly
  curved imported-like face is fold-free, both with area within tolerance of a dense reference.
- No boundary vertex moves (watertightness preserved); OCC oracle green; lint clean.
- Live confirmation on EDF (shaded + Normal-Debug): the staircase is gone, surfaces read smooth.

## Depends on

`trimmedPatchMesh` (the watertight baseline), F01 (pcurve, for the boundary), PBI-315 (fold repair),
the OCC oracle. Reuses the merged PBI-314 adaptive-density idea (re-targeted to 3D deviation).
