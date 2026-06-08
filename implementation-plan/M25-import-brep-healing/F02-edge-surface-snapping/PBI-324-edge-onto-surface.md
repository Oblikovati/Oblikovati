---
milestone: M25
feature: F02
pbi: PBI-324
title: Snap edge curves onto their faces' surfaces within tolerance
status: planned
estimate: M
---

# PBI-324 — Snap edge curves onto their faces' surfaces within tolerance

**Milestone:** M25 Imported B-Rep Healing  ·  **Feature:** F02 Edge/surface tolerance snapping

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
