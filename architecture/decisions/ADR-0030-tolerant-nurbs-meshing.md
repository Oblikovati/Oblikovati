# ADR-0030 — Tolerant NURBS surface meshing (on-surface interior nodes + shared-edge stitching)

**Status:** Accepted (2026-06-08) — milestone [M24](../../implementation-plan/M24-tolerant-nurbs-meshing/_milestone.md).
F01 (on-surface pcurves) in progress.
**Context:** Tessellating trimmed freeform (B-spline/NURBS) faces of *imported* B-reps.
**Builds on / refines:** [ADR-0027](ADR-0027-curved-face-boolean.md) and the curved-face
tessellation chain in `kernel/ops/{tessellate_trim,cdt,refined_patch}.go` (PRs #100–#105).

## Problem

A trimmed NURBS face must mesh **smooth** (no fold), **volume-correct** (the divergence-theorem
mass-properties must match the true solid), and **watertight** with its neighbours. The shipped
path (`trimmedPatchMesh`) triangulates the face from its **boundary loops only**. A boundary-only
triangulation of a *curved* face cannot follow the 3D curvature, so it **folds** — the diagonal
"staircase" crease seen on EDF.STEP's external faces.

Interior nodes are required to follow the curvature, but every naive attempt inflated the EDF
volume by +20–70% (measured against OpenCASCADE `getMass` = 207,002):

| approach | EDF volume |
|----------|-----------:|
| `(u,v)` boundary-only (today) | 210,087 (101.5%, folds) |
| best-fit-plane boundary-only | ~159,660 (folds + tears) |
| `(u,v)` + interior grid | ~276,548 (+33%) |
| structured grid over domain | ~303,800 (+47%) |
| march-pcurve + interior | ~279,921 (+35%) |

Two root causes, both confirmed:

1. **Imported edge curves lie ~mm off their own surfaces** (a STEP authoring tolerance, not a bug
   — `Surface.ParamAt` correctly returns the *closest* surface point; a Gauss–Newton projection
   step gave the identical residual). So a node placed **on** the surface does not coincide with
   the **off-surface** shared edge boundary.
2. **No single 2D embedding is conformal for every face**: the surface's own `(u,v)` self-intersects
   from `ParamAt` jitter near folds; a best-fit plane self-intersects on large wrapping patches. An
   interior `(u,v)` grid + CDT then **over-encloses** on complex/holed trims.

## Decision

Mesh each freeform face **on its own surface** with **deflection-adaptive interior nodes**, and
make adjacent faces meet by **stitching to a shared edge polyline within tolerance** — the approach
OpenCASCADE's `BRepMesh` takes. Concretely:

### 1. On-surface pcurves, march-projected (F01)

Compute each loop's boundary `(u,v)` as a **pcurve**: project the first point with a full grid
search, then each subsequent point seeded from the *previous* point's `(u,v)` (`BSplineSurface.ParamNear`),
so the curve stays on one smooth branch. This eliminates the self-intersection that independent
`ParamAt` produces near folds — the boundary is smooth and non-self-intersecting, the prerequisite
for a non-folding triangulation and a reliable point-in-trim test.

### 2. Trim-respecting, deflection-adaptive interior + curvature-aware triangulation (F02)

Generate interior `(u,v)` nodes whose density follows local curvature (`ops.Quality`), kept
**strictly inside** the trim (inside the outer pcurve, outside every hole pcurve, with a margin) so
they never spill and over-enclose. Triangulate (pcurve boundary + interior) with the existing CDT,
then **detect and repair folds** (adjacent triangles whose 3D normals oppose) by local re-triangulation
or added density, so the lifted mesh does not double back.

### 3. Tolerant shared-edge stitching (F03)

Mesh each topological **edge once** as a single 3D polyline (`discretizeEdge`, already shared by
both faces). Both faces use those exact points as their **boundary vertices** while meshing their
interior on their own surface; the stitch band spans the ~mm edge/surface offset uniformly. The
body stays watertight even though no face's surface passes exactly through the edge.

### 4. Oracle-gated, no regressions (F04)

Every step is gated by the OpenCASCADE volume oracle plus committed **fold** and **over-enclosure /
per-face-area** detectors. A per-face inflation guard falls back to the boundary-only path for any
face that would over-enclose, so the milestone never ships a volume regression.

## Consequences

- Freeform faces render smooth and stay volume-correct; the staircase fold is removed.
- The boundary moves the ~mm tolerance off the literal edge curve onto the shared polyline; this is
  the accepted import tolerance, distributed evenly, not a visible gap.
- This is a meshing concern only — it does not heal the underlying B-rep (edges still lie off
  surfaces in the model); a separate import-healing effort could remove the tolerance at the source.
- Analytic faces (planes, quadrics, sphere caps) are unaffected — they already mesh correctly via
  `structuredGridMesh` / `gridPatchMesh`.
