---
milestone: M22
feature: F10
pbi: PBI-243
title: Surface↔curve intersection + point projection to surface
status: planned
estimate: L
---

# PBI-243 — Surface↔curve intersection + point projection

**Milestone:** M22  ·  **Feature:** F10 Surface-Intersection Kernel

## Goal
Compute where a 3D curve crosses a surface, and the closest point of a surface to a
given point (foot of perpendicular) — the kernel `ProjectToSurfaceCurve`/`OnFaceCurve`
need.

## Scope / work
- `kernel/geom/project.go`: `ClosestPointOnSurface(s Surface, p Point3) (u,v, foot)` via
  Newton on ∇‖S(u,v)−p‖² with subdivision seeding; closed-form for plane/sphere/cylinder.
- `kernel/geom/intersect_surface_curve.go`: `IntersectCurveSurface(c Curve3, s Surface)
  []Point3` — sample the curve, bracket sign changes of the surface implicit/foot
  distance, refine by bisection+Newton.
- Tolerances + max-iteration guards in `geom_internal.go`.

## Acceptance criteria
- Property tests ≥98%: every returned point lies on both inputs to 1e-7; projection foot
  is perpendicular (residual tangent·(foot−p)=0); metamorphic — refining the sampling
  does not change the root set.
- Closed-form cases match analytic answers (line∩plane, line∩sphere).
- `make ci` green (CGO off).

## Depends on
`kernel/geom` surfaces.
