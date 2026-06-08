---
milestone: M24
name: Tolerant NURBS Surface Meshing (imported B-rep)
status: planned
---

# M24 — Tolerant NURBS Surface Meshing (imported B-rep)

Mesh trimmed **freeform (B-spline/NURBS) faces** of imported B-reps so curved external
surfaces render **smooth, fold-free, and volume-correct**. Today
[`kernel/ops/tessellate_trim.go`](../../kernel/ops/tessellate_trim.go) meshes a non-rectangular
NURBS face from its **boundary loops only** (`trimmedPatchMesh` → the CDT in
[`cdt.go`](../../kernel/ops/cdt.go)). A boundary-only triangulation of a *curved* face cannot
follow the 3D curvature, so it **folds** — the diagonal "staircase" crease seen on EDF.STEP's
external faces (15 fold-edges on body-0 faces 0/5). Interior nodes are required, but every
naive attempt **inflated the divergence-theorem volume +20–70%** on the EDF model (see the
diagnosis in `step-import-status` memory + the table below). This milestone builds the real
tolerant mesher — the class of algorithm OpenCASCADE's `BRepMesh` implements — incrementally,
gating **every** step with the OpenCASCADE `getMass` volume oracle so no step ships a regression.

This is the deferred follow-on to the curved-face tessellation chain (#100 winding, #101 (u,v)
ear-clip, #102 CDT, #103 sphere-cap band, #105 sphere-cap interior nodes), which brought EDF
import from ~75% to **101.5%** of the OpenCASCADE volume and made analytic caps smooth. The one
remaining defect is the freeform external faces folding.

## Why the naive fixes failed (all measured on EDF.STEP; OCC truth = 207,002)

| approach | EDF volume | result |
|----------|-----------:|--------|
| `(u,v)` boundary-only (today) | 210,087 (101.5%) | volume OK, **folds** (the staircase) |
| best-fit-plane boundary-only | ~159,660 (75%) | folds **and** tears |
| `(u,v)` + interior grid | ~276,548 (133%) | **over-encloses** |
| structured grid over domain | ~303,800 (147%) | folds **and** over-encloses |
| march-projected pcurve + interior | ~279,921 (135%) | big faces fold-free, still over-encloses |

Two root causes, both confirmed:
1. **Imported edge curves lie ~1.9 mm off their own surfaces** (a STEP tolerance reality;
   `Surface.ParamAt` correctly returns the *closest* surface point — verified not a solver bug, a
   Gauss–Newton step gave the identical residual). So on-surface interior nodes do not align with
   the off-surface shared boundary → overlap.
2. **No single 2D embedding works for every face** (the surface's own `(u,v)` self-intersects
   from `ParamAt` jitter near folds; the best-fit plane self-intersects on big wrapping patches),
   and an interior `(u,v)` grid + CDT **over-encloses** on complex/holed trims.

The march-projected pcurve (project each edge point seeded from the previous point's `(u,v)`,
`ParamNear`) **does** produce a smooth, non-self-intersecting boundary and fixed the big duct
faces' folds — it is the foundation this milestone builds on.

## Goals

- Trimmed NURBS faces mesh **smooth** (no fold/staircase) and **volume-correct** (within the OCC
  oracle tolerance), with **interior** nodes following the surface.
- A **tolerant shared-edge** representation so adjacent faces meet without gaps, folds, or
  double-counting — the watertightness that survives the ~mm edge/surface import tolerance.
- **Deflection-adaptive** node density (the OpenCASCADE range-splitter idea) so flat regions are
  coarse and high-curvature regions are dense, driven by `ops.Quality`.
- Every step **gated by the volume oracle** and by committed fold / over-enclosure detectors —
  no eyeballing, no regressions.

## In scope

- Smooth per-edge **pcurves** computed by march-projection (`ParamNear`), shared by both
  adjacent faces — the on-surface, non-self-intersecting boundary.
- **Trim-respecting** interior node generation: a deflection-adaptive `(u,v)` grid that stays
  strictly inside the trim (inside outer, outside every hole) with no spill.
- **Curvature-aware** face triangulation (CDT of boundary + interior) that does not fold in 3D.
- **Tolerant shared-edge stitching**: mesh each edge once (one polyline), both faces bind to it,
  so the body stays watertight despite the edge/surface tolerance.
- Fold-freeness + over-enclosure + per-face-area committed detectors; OCC oracle extended with a
  freeform-NURBS fixture; the EDF model as an end-to-end volume regression.

## Out of scope (handled elsewhere / deferred)

- **Analytic** face meshing (planes, cylinders, cones, tori, sphere caps) — already correct
  (`structuredGridMesh` / `coneApexFan` / `gridPatchMesh`); M24 touches only the freeform path.
- **Exact analytic B-rep operations** on NURBS (boolean, offset) — geometry, not meshing.
- **Re-projecting / healing** the imported B-rep to remove the edge/surface gap at the model
  level (a separate import-healing effort); M24 tolerates the gap in the mesher.
- Reading STEP **pcurves** when present — irrelevant for the driving case (SolidWorks writes
  `0 SURFACE_CURVE`); M24 computes pcurves by projection. (A future optimization may consume them.)

## Exit criteria

- EDF.STEP imports with the external freeform faces **fold-free** (the committed fold detector
  reports 0 fold-edges on every face) and the total volume stays within the oracle tolerance of
  OpenCASCADE's `getMass` (no inflation; target ≤ a few % of 207,002).
- A synthetic **trimmed freeform-NURBS** OCC fixture (a B-spline patch with a hole) meshes within
  volume tolerance and fold-free, committed in the oracle suite.
- Adjacent freeform faces share an edge polyline: no per-face gap exceeds the model tolerance
  (a watertightness/free-edge check on the assembled body).
- Live confirmation on EDF (shaded + Normal-Debug): the staircase is gone, the external surfaces
  read smooth — using the Save-Viewport-PNG loop.
- All existing tessellation tests + the OCC oracle stay green at every PBI; lint clean.

## Depends on

M07 (B-rep topology + tessellation, `kernel/ops`), M17 (STEP import, `kernel/exchange/step`),
the curved-face chain in `kernel/ops/{tessellate_trim,cdt,refined_patch}.go`, the OCC oracle
(`kernel/exchange/step/occ_oracle_test.go` + gmsh SDK ground truth). A new
**ADR-0030 (tolerant NURBS meshing)** records the on-surface-with-tolerance approach (F01).

## Features

| ID | Feature | PBIs | Summary |
|----|---------|:----:|---------|
| **F01** | [On-surface pcurves & boundary](F01-pcurves-on-surface-boundary/_feature.md) | 2 | `ParamNear` seeded projection + march-projected per-edge pcurves (smooth, non-self-intersecting on-surface boundary); the ADR. |
| **F02** | [Trim-respecting adaptive interior](F02-trim-respecting-interior/_feature.md) | 3 | Deflection-adaptive interior `(u,v)` nodes clipped strictly to the trim (the over-enclosure fix) + curvature-aware non-folding CDT. |
| **F03** | [Tolerant shared-edge stitching](F03-shared-edge-stitching/_feature.md) | 2 | Mesh each edge once; both faces bind to the shared polyline so the body stays watertight across the ~mm edge/surface tolerance. |
| **F04** | [Oracle gating & EDF regression](F04-oracle-gating-regression/_feature.md) | 2 | Committed fold / over-enclosure / per-face-area detectors, a synthetic trimmed-NURBS OCC fixture, and EDF as an end-to-end volume regression. |
