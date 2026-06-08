---
milestone: M25
feature: F02
pbi: PBI-324
title: Snap edge curves onto their faces' surfaces within tolerance
status: done
estimate: M
---

# PBI-324 — Snap edge curves onto their faces' surfaces within tolerance

**Milestone:** M25 Imported B-Rep Healing  ·  **Feature:** F02 Edge/surface tolerance snapping

## DONE (2026-06-08)

`kernel/ops/snap_edges.go` — `SnapEdgesToSurfaces(b, q)` snaps every edge onto its adjacent
surfaces and stores the result on the edge (`topo.Edge.SnappedCurve` / `Tolerance`, new), which
`discretizeEdge` now returns verbatim so **both faces mesh the identical boundary**. Snapping uses
the surfaces' `ParamAt` inverse (`geom.ProjectCurveToSurface`), not the geometric foot, on purpose:
the grid mesher reproduces `s.PointAt(s.ParamAt(p))` and `ParamAt∘PointAt` is the identity on
parameters, so a `ParamAt`-snapped point is the exact fixed point the grid re-samples.

Reconciliation is keyed on **grid-meshed vs boundary-verbatim**, not analytic vs B-spline: a plane
(earcut) and a B-spline (pcurve mesher) take the 3D boundary verbatim, while cylinder/cone/sphere/
torus re-sample via `ParamAt`. So the snap lands on the grid neighbour (which constrains it); two
grid faces converge to their intersection; two verbatim faces take the midpoint. Edges already on
their surfaces (residual < `snapResidualFloor` = 1e-6 — every OpenCASCADE fixture) are left native,
so accurate imports and modelled solids do not move. Endpoints are **not** pinned (a closed circle's
seam would kink back to its off-surface vertex); welding the shared corners that incident snapped
edges now disagree on is **PBI-325** (vertex merge).

Tests (`kernel/ops/snap_edges_test.go`): off-surface rim lands on the cylinder; a 128→0 free-edge
watertightness regression; clean solid left native + still watertight; two-grid intersection; the
grid-preference policy; identical-boundary; idempotence. OCC oracle + full kernel suite green; lint
clean.

**Verified on EDF then UNWIRED (2026-06-08):** `ops.HealImportedBody` (`kernel/ops/heal_import.go`,
snap → ReconstructPcurves in that order) was wired into `model/feature.ImportBodies`, verified on EDF,
then **reverted** — healing is a no-op on EDF (see finding) so it added an unproven transform to the
global import path for no benefit. `HealImportedBody`/`SnapEdgesToSurfaces` stay as tested building
blocks (like the also-unwired `ReconstructPcurves`), to be wired by the M25 heal-entry orchestration
once the PBI-330 mesher fix actually moves EDF.

**KEY FINDING — edge snapping does NOT fix EDF watertightness; this PBI's premise (and PBI-330's) was
wrong about the cause.** Measured EDF edge-off-surface residuals are ~0.3 µm (median), i.e. the imported
edges are essentially ON their surfaces; the "0.017–8.56 mm" figure in PBI-330 was the free-edge PARTNER
gap (unpaired mesh edges), a different quantity. A gating sweep on EDF body3 (69 free edges baseline):
snapping everything → 75 (WORSE), snapping analytic-only → 69 (no change). The regression came from
snapping B-spline-adjacent edges, which pulls a freeform boundary off its surface and folds the NURBS
metric CDT. So `snapEdge` now LEAVES B-spline-adjacent edges native, and healing is a verified no-op on
EDF (free edges per body unchanged: 4/0/8/69/0/0 before==after). EDF's leaks are mesher-internal (the
NURBS CDT interior + grid-vs-CDT boundary sampling), i.e. the real **PBI-330 / NURBS-watertightness**
work — NOT edge snapping. Snapping remains correct and active for genuinely off-surface ANALYTIC imports
(the synthetic 128→0 case). Also note: `TessellateBody` free-edge counts are non-deterministic on some
imported solids (filleted_box 36↔29 unmutated) — a separate latent bug.

## Goal

Make each edge's discretized boundary lie on its adjacent surfaces (within tolerance) so the two
faces of an edge mesh the same points — removing the ~1.9 mm off-surface gap.

## Scope / work

- For each edge shared by faces A and B, derive its boundary samples from the pcurves
  (`A.PointAt(pcurveA)`, `B.PointAt(pcurveB)`) and reconcile to a single shared polyline (e.g. the
  average, or A's, recording the max deviation as the edge tolerance). Use that one polyline for
  both faces' tessellation boundary.
- Record the residual (the import gap) as the edge's tolerance; surface it for validation.
- Leave well-formed edges (already on-surface, tiny residual) untouched.

## API contracts (interfaces / enums / collections)

- (internal) shared edge polyline + per-edge tolerance in the heal path.

## Acceptance criteria

- EDF.STEP: after snapping, an edge's polyline lies on BOTH adjacent surfaces within the recorded
  tolerance (committed check); the two faces use identical boundary points.
- A clean OCC fixture (edges already on surfaces) is unchanged (residual ~0, no movement).
- OCC oracle green.

## Depends on

PBI-323 (pcurves).
