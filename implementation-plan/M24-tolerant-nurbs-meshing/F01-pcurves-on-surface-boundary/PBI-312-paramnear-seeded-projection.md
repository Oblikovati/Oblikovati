---
milestone: M24
feature: F01
pbi: PBI-312
title: ParamNear seeded surface projection + ADR
status: done
estimate: S
---

# PBI-312 — `ParamNear` seeded surface projection + ADR

**Milestone:** M24 Tolerant NURBS Surface Meshing  ·  **Feature:** F01 On-surface pcurves & boundary

## Goal

Add a seeded closest-point projection on a NURBS surface so a marching caller can keep a curve on
a single smooth branch, and record the milestone's approach in an ADR.

## Scope / work

- `func (s BSplineSurface) ParamNear(q math.Point3, u0, v0 float64) (u, v float64)` in
  `kernel/geom/paramat.go`: start from `(u0,v0)` (clamped to domain), run the existing
  `surfaceProjectStep` to convergence — no `surfaceGridSeed`. Reuse the existing step.
- Write **ADR-0030 — Tolerant NURBS surface meshing**: imported edge curves lie ~mm off their
  surfaces (a tolerance reality, not a bug); we mesh each face *on its surface* with interior
  nodes and **stitch shared edges within tolerance** rather than re-project the off-surface
  boundary (which inflates volume). Records the measured failure table from the milestone.

## API contracts (interfaces / enums / collections)

- `geom.BSplineSurface.ParamNear(q, u0, v0) (u, v float64)` — internal kernel API.

## Acceptance criteria

- For points sampled **on** a known B-spline surface, `ParamNear` seeded from a nearby `(u,v)`
  returns a `(u,v)` whose `PointAt` round-trips to the sample within tight tolerance.
- For two nearby points straddling a near-fold, `ParamNear` (seeded from the same prior `(u,v)`)
  returns **nearby** `(u,v)` for both, where independent `ParamAt` returns far-apart `(u,v)`
  (the branch-jump). Unit-tested on a constructed folded patch.
- ADR-0030 committed and linked from the milestone.
- `go test ./kernel/geom/...` green; lint clean.

## Depends on

`kernel/geom/paramat.go`.
