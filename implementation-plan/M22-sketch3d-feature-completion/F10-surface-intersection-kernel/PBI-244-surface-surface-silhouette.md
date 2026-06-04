---
milestone: M22
feature: F10
pbi: PBI-244
title: Surface↔surface intersection + silhouette extraction
status: done
estimate: L
---

# PBI-244 — Surface↔surface intersection + silhouette

**Milestone:** M22  ·  **Feature:** F10 Surface-Intersection Kernel

## Goal
Trace the intersection curve(s) of two surfaces, and the silhouette of a surface for a
given view/projection direction — the kernel `IntersectionCurve`/`SilhouetteCurve` need.

## Scope / work
- `kernel/geom/intersect_surface_surface.go`: marching-on-both-surfaces tracer (seed
  via subdivision overlap, step along the cross product of the two surface normals,
  re-project each step onto both surfaces); returns polyline(s) fit to a `BSplineCurve`.
  Closed-form for plane∩plane, plane∩cylinder/cone/sphere where available.
- `kernel/geom/silhouette.go`: `Silhouette(s Surface, dir Vector3) []Curve3` — trace
  the locus where the surface normal ⟂ view direction (n·dir = 0).

## Acceptance criteria
- Property tests ≥98%: traced points lie on both surfaces to 1e-6; closed analytic cases
  (plane∩plane = line, plane∩sphere = circle) match; silhouette points satisfy n·dir≈0;
  metamorphic under step refinement.
- `make ci` green (CGO off).

## Depends on
PBI-243.
