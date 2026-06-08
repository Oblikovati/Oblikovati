---
milestone: M24
feature: F03
pbi: PBI-318
title: Watertight assembled body across the edge/surface tolerance
status: planned
estimate: M
---

# PBI-318 — Watertight assembled body across the edge/surface tolerance

**Milestone:** M24 Tolerant NURBS Surface Meshing  ·  **Feature:** F03 Tolerant shared-edge stitching

## Goal

Guarantee the assembled body mesh is watertight (no free-edge gaps beyond tolerance) despite the
imported edge/surface offset, now that every face binds to shared edge polylines.

## Scope / work

- A body-level free-edge / manifold check over the merged per-face meshes: every shared edge is
  used by exactly two faces and its vertices coincide; report any gap exceeding the model
  tolerance.
- Reconcile any remaining mismatch (e.g. a degenerate or seam edge) — weld coincident shared
  vertices; flag genuinely non-manifold input rather than silently gapping.
- Confirm mass-properties volume is unaffected (the per-face meshes already share edge points).

## API contracts (interfaces / enums / collections)

- (internal) body watertightness/free-edge check (test helper + optional debug surface).

## Acceptance criteria

- EDF.STEP assembled body: no free-edge gap exceeds the model tolerance (committed check);
  free-edge count equals the genuine open-boundary count (0 for the closed solids).
- Volume within oracle tolerance of OCC; OCC oracle green.
- Live: no cracks visible between freeform faces (shaded, Save-Viewport-PNG).
- `go test ./kernel/...` green; lint clean.

## Depends on

PBI-317.
