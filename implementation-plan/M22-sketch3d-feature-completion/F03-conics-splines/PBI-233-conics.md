---
milestone: M22
feature: F03
pbi: PBI-233
title: 3D ellipse + elliptical arc entities + tools
status: done (model+API; UI in F12)
estimate: S
---

# PBI-233 — 3D ellipse + elliptical arc

**Milestone:** M22  ·  **Feature:** F03 Conics & Splines

## Goal
Add `SketchEllipse3D` and `SketchEllipticalArc3D` over `kernel/geom.EllipseFull`/
`EllipticalArc`, creatable through `/api` + tool.

## Scope / work
- `model/sketch/conics_3d.go`: the two entities (center point + major-axis dir +
  major/minor radii; arc adds start/end angle).
- `/api`: `Sketch3DEntityEllipse/EllipticalArc`; `AddSketch3DEntityArgs` conic fields
  reused; `client` helpers.
- router case; UI tool + ribbon button.

## Acceptance criteria
- Unit ≥98%; dogfood add+enumerate; round-trip; ≥1 UI e2e test; `make ci` green.

## Depends on
PBI-232.
