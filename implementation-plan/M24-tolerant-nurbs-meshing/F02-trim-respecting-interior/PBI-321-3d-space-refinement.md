---
milestone: M24
feature: F02
pbi: PBI-321
title: 3D-space interior refinement (replaces the (u,v) grid)
status: blocked
estimate: L
---

> **BLOCKED (2026-06-08): the imported surfaces are unsamplable, not just non-conformal.**
> Implemented 3D-space refinement — split bulging interior edges, projecting the chord MIDPOINT
> onto the surface (`PointAt(ParamAt(mid))`). On EDF it was WORSE: 1369 folds, +33% volume. The
> midpoints lie between trim points, yet `ParamAt(midpoint)` STILL lands far off — the imported
> rational-NURBS surfaces are pathological (non-conformal AND `ParamAt` finds a distant closest
> point), so **no interior sampling via the surface is reliable**. Even edge-flip repair on the
> boundary-only mesh (no new samples) only goes 13→12 folds — the fold is inherent to a
> boundary-only triangulation of a curved face and there are no trustworthy interior points to
> flip with. **Only the edge curves (the boundary) are reliable; the surface interior is not.**
> This is an **import-quality** blocker, not a mesher one. M24 cannot mesh these faces fold-free
> from the available data. The real fix is **import healing** — reparameterize the imported NURBS
> (or compute reliable pcurves) so the surface can be sampled — a separate effort upstream of the
> mesher. F01 (pcurves) + PBI-314 (adaptive density) + PBI-315 (fold repair) stay merged + tested,
> ready to use once the surface is reliable. Reverted to the boundary-only baseline (210k).

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
