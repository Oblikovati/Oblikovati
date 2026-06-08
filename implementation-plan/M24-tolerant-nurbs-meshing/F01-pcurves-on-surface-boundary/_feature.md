---
milestone: M24
feature: F01
name: On-surface pcurves & boundary
status: in-progress
---

# M24 · F01 — On-surface pcurves & boundary

The foundation: a **smooth, non-self-intersecting `(u,v)` boundary on the surface** for a
freeform face. Today the boundary `(u,v)` comes from calling `Surface.ParamAt` independently per
edge point; near a fold those calls snap to different local minima, so the boundary
**self-intersects** — which makes a boundary-only triangulation fold and an interior grid
over-enclose. The fix is to compute each edge's **pcurve** by march-projection: project the first
point with the full grid search, then each subsequent point seeded from the previous point's
`(u,v)` (`ParamNear`), so the curve stays on one smooth branch.

This was prototyped (and reverted) during diagnosis; it demonstrably produced a smooth boundary
and fixed the big duct faces' folds. F01 lands it properly with isolation tests and the ADR. It
is infrastructure — on its own it does not change the mesh (boundary-only still folds on curved
faces); F02/F03 consume it.

## In scope

- `geom.BSplineSurface.ParamNear(q, u0, v0)` — seeded surface projection (no grid search).
- `ops` march-projected pcurve builder for a face's boundary loops (outer + holes).
- A test that the resulting `(u,v)` boundary is **non-self-intersecting** where independent
  `ParamAt` self-intersects.
- ADR-00XX recording the on-surface-with-tolerance meshing approach.

## Out of scope

- Interior node generation (F02) and shared-edge stitching (F03).
- Reading STEP-supplied pcurves (absent in the driving case).

## Key API contracts delivered

- `geom.BSplineSurface.ParamNear` (internal kernel API).
- (internal) `ops` pcurve builder.

## Depends on

`kernel/geom/paramat.go` (`surfaceProjectStep`, `surfaceGridSeed`), `kernel/ops/tessellate_trim.go`.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-312](PBI-312-paramnear-seeded-projection.md) | `ParamNear` seeded surface projection + ADR |
| [PBI-313](PBI-313-march-projected-pcurves.md) | March-projected per-edge pcurves (non-self-intersecting boundary) |
