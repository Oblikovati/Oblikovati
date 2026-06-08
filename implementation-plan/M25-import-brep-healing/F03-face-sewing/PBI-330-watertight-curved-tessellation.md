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

## UPDATE (2026-06-08): test-first narrowed the fix to edge snapping (PBI-324), not mesher rewrite

A committed test (`TestCleanCurvedSolidIsWatertight`, kernel/ops) proves a **clean** curved solid
tessellates **watertight (0 free edges)** — the grid meshers already share boundaries when the edge
lies on the surface (a face's `PointAt(ParamAt(edge))` equals the shared edge point). So the meshers
are NOT buggy; the EDF leak is purely that imported edges sit ~mm OFF their surfaces, so the projected
boundary diverges from the shared edge point. Free-edge partner gaps measured 0.017–8.56 mm (real, not
weldable). **Therefore the fix is [PBI-324 edge snapping](../F02-edge-surface-snapping/PBI-324-edge-onto-surface.md)**
— snap imported edges onto their surfaces so the proven-watertight meshers work — NOT conforming the
meshers (this PBI). Keep this PBI only as a fallback if snapping can't reconcile an edge that's off
BOTH adjacent surfaces. Status: superseded-by PBI-324 for the EDF case.

## CORRECTION (2026-06-08, after PBI-324 shipped + was verified on EDF): the premise above was WRONG

PBI-324 (edge snapping) is **done, wired into import, and DOES NOT fix EDF** — reopening this PBI as the
real fix. The "imported edges sit ~mm off their surfaces" claim conflated two quantities: the **free-edge
partner gap** (0.017–8.56 mm, the distance between unpaired mesh edges) is NOT the **edge-off-surface
residual** — measured directly, EDF's edges sit ~0.3 µm (median) ON their surfaces. Snapping them
therefore does nothing for watertightness (EDF body3 stayed 69 free edges), and snapping the B-spline-
adjacent ones made it WORSE (69→75) by folding the NURBS CDT. **So the EDF leak IS mesher-internal**, as
this PBI originally said: (1) the grid meshers (`structuredGridMesh`/`gridPatchMesh`) vs the NURBS CDT
sample shared edges differently, and (2) the NURBS metric CDT itself leaves T-junctions/folds on body3.
The fix is this PBI's "conform the meshers to the shared `discretizeEdge` boundary" OR a mesh-sew
post-process — independent of edge snapping. (Also: `TessellateBody` free-edge counts are
non-deterministic on some imported solids — fix that first so the metric is reproducible.)

## Phased plan + progress (2026-06-08, after determinism fix landed)

Prerequisite DONE: `TessellateBody` is now deterministic (CDT cavity + fold-repair map order), so
free-edge counts are stable, measurable numbers. Per-face diagnosis split the leaks into two
mechanisms, tackled in phases:

- **Phase 1 — closed-surface watertightness (DONE).** A bare closed surface (OCC's whole sphere = 1
  face, 0 seam edges) fell to the full-domain grid, which duplicated the periodic seam (u=0≡2π → welded
  degree-4 edges) and degenerated the poles (zero-area quads → degree-32). `kernel/ops/closed_surface_mesh.go`
  (`closedDomainMesh`, replacing the naive `gridMesh` in `fullDomainGridMesh`) wraps the seam onto the
  first column and shares one vertex per pole row, fanning its ring. **sphere 66→0 free edges**, area
  ≈4πr²; tests in `closed_surface_mesh_test.go`; OCC oracle + full kernel suite green.
- **Phase 2a — periodic band seam (DONE).** A full cylinder/cone side (`periodicBandGrid`) dropped one
  cell at the seam: the seam vertex's angle read back as ~2π−ε (tiny negative coordinate), so `us`
  landed only at ~2π with no 0 column → 31 cells for a 32-segment circle → a one-cell crack against the
  caps (the leak appears one cell in, not at the seam, because the cap polygon and band grid misalign by
  one). Fix: `bracketPeriod` snaps the seam sample to 0 and brackets a closed [0, 2π], and the band now
  routes through `closedDomainMesh` (wraps the seam onto one column). **cylinder 4→0, cone_frustum 0,
  cone_sharp 33→0** — the apex cone too (NOTE: this also subsumed the planned Phase 3). Edges are
  exactly on-surface (residual ~3e-15 mm) — this is a grid-conformance bug, not off-surface.
- **Phase 2b — sphere-cap / gridPatchMesh robustness (DONE).** filleted_box 36 was the sphere corner-
  fillet caps whose `(u,v)` reaches the pole (v=±π/2, all u collapse) or wraps the seam: the CDT tore
  them into a non-manifold mesh (interior holes / pole overlaps, visible only after a 3D weld). The cap
  boundary was already the exact shared edge points, so the fix is robustness, not conformance:
  `gridPatchMesh` now checks the result (`patchIsManifold` = welded free edges ≤ loop boundary) and
  falls back to the plane-based `boundaryPatchMesh` (watertight, boundary-only) when the CDT tore. Only
  torn caps fall back; smooth caps (and the refinement test patch) keep their interior nodes.
  **filleted_box 36→0; EDF total 81→44** (the fallback fixed EDF's pole-degenerate caps too). Guard:
  filleted_box added to `TestImportedAnalyticPrimitivesWatertight` + `weldedFreeEdgeCount`/
  `patchIsManifold` unit tests.
- **Phase 4 — remaining EDF leaks (IN PROGRESS; 44→36).** Diagnosis: the remaining leaks are NOT mainly
  NURBS — they are trimmed analytic walls + a mixed grid↔NURBS↔plane core.
  - **Done (body2 8→0):** a non-iso-rectangular trimmed cylinder/cone wall fell to `boundaryPatchMesh`
    (best-fit-plane ear-clip), which tears a curved wall that lies in no plane. `nonRectangularMesh` now
    routes cyl/cone to `metricWallMesh` — a CDT in METRIC-SCALED (u,v) (√E,√G, generalising the NURBS
    metric mesher; `metricScale` now clamps the cylinder's infinite v-domain). Isotropic interior grids
    explode on a cylinder's anisotropic (u,v); the metric scaling + deflection-bounded interior fixes
    both correctness and the node-count blow-up. EDF total 44→36; all OCC fixtures still 0.
  - **Remaining (body3 32, body0 4):** distributed across band/iso-rectangle cyl/cone (`structuredGridMesh`
    re-sampling its own grid lines, not the shared `discretizeEdge`) + B-spline + plane edges. Likely
    needs the band/iso-rectangle grid to emit its boundary ROW as the exact edge points, OR a
    `TessellateBody` mesh-sew (weld + zip T-junctions) for the mixed remainder. Hardest; next.

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
