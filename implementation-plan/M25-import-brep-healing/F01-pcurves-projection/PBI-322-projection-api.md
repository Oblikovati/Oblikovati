---
milestone: M25
feature: F01
pbi: PBI-322
title: Robust point/curve-to-surface projection API
status: done
estimate: M
---

# PBI-322 — Robust point/curve-to-surface projection API

**Milestone:** M25 Imported B-Rep Healing  ·  **Feature:** F01 Robust projection & pcurve reconstruction

## Goal

Promote the verified surface projection into a first-class, reusable API the healing passes build on:
a point inversion (closest `(u,v)`) and a curve-on-surface projection (a continuous pcurve).

## Scope / work

- `ProjectPointToSurface(s, p) (u, v, dist)`: multi-seed (coarse grid) + Newton/Gauss-Newton (the
  verified `surfaceProjectStep`), returning the global-closest `(u,v)` and the residual distance.
  Handle periodic domains (wrap) and report the residual so callers know the off-surface gap.
- `ProjectCurveToSurface(s, pts) []Point2`: march along an ordered curve sampling, each point seeded
  from the previous (`ParamNear`), giving a continuous, non-self-intersecting pcurve (generalises
  `ops.marchUV` into `geom`/a shared package).
- M24 already proved `ParamNear` ≈ brute-force; this PBI is API shape + periodicity, not new numerics.

## API contracts (interfaces / enums / collections)

- `ProjectPointToSurface`, `ProjectCurveToSurface` (kernel geometry API).

## Acceptance criteria

- On constructed surfaces (plane, cylinder, sphere, a curved B-spline), point inversion round-trips
  on-surface points to tolerance and returns the residual for off-surface points; matches a
  brute-force dense-grid closest point within tolerance.
- A periodic (cylinder/sphere-longitude) projection returns `(u,v)` in domain without seam jumps.
- `ProjectCurveToSurface` of a sampled edge is continuous (no `(u,v)` jumps) and non-self-intersecting.
- Unit-tested; lint clean.

## Depends on

M24 `ParamNear`/`surfaceProjectStep` (verified), `kernel/geom`.
