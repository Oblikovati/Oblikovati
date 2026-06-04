---
milestone: M22
feature: F11
pbi: PBI-245
title: IntersectionCurve + OnFaceCurve + ProjectToSurfaceCurve
status: planned
estimate: L
---

# PBI-245 — Intersection / on-face / project-to-surface curves

**Milestone:** M22  ·  **Feature:** F11 Surface-Derived Curves

## Goal
Expose the three reference-bound surface curves as recompute-driven 3D-sketch entities.

## Scope / work
- `model/sketch/surface_curves_3d.go`: `IntersectionCurve` (two face refs → F10
  surface∩surface), `OnFaceCurve` (a 2D-on-face curve → 3D via the face's surface),
  `ProjectToSurfaceCurve` (curve ref + face ref → F10 projection). Each bound by
  reference key; recompute re-evaluates the kernel; HealthStatus on lost refs.
- `/api`: entity kinds + `AddSketch3DSurfaceCurveArgs` (face/curve refs, direction);
  collections (`IntersectionCurves`, `OnFaceCurves`, `ProjectToSurfaceCurves`);
  `client` helpers.
- router cases; UI tools + ribbon buttons.

## Acceptance criteria
- Unit ≥98%: against analytic faces (cylinder∩plane = ellipse/circle; project a line
  onto a plane = line); identity tests (survive recompute / fail honest / reload).
- Dogfood; round-trip; ≥1 UI e2e test per tool; `make ci` green.

## Depends on
PBI-243, PBI-244, PBI-241.
