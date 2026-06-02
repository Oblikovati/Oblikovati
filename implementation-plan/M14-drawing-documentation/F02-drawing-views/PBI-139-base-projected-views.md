---
milestone: M14
feature: F02
pbi: PBI-139
title: Base & projected views with hidden-line
status: planned
estimate: XL
---

# PBI-139 — Base & projected views with hidden-line

**Milestone:** M14 Drawing & Documentation  ·  **Feature:** F02 Drawing Views

## Goal

Implement base-view creation from a 3D model with orientation/scale/rendering-style and projected (orthographic) views, including the hidden-line computation that produces visible/hidden drawing curves.

## Scope / work

- `DrawingViews.AddBaseView`/`AddProjectedView`.
- Hidden-line engine → `DrawingCurve(s)` visible/hidden.
- Rendering style (wireframe/hidden/shaded).
- `DrawingCurve`↔model-edge reference keys.

## API contracts (interfaces / enums / collections)

- `DrawingView(s)`,`ProjectedDrawingView`,`DrawingCurve(s)`,`DrawingCurveSegment(s)`

## Acceptance criteria

- A base+projected pair shows correct visible/hidden edges and stays associative on model edits.

## Depends on

_See feature dependencies._

## Notes

Hidden-line removal from B-rep is a substantial engine. View↔model associativity depends on M03 reference keys.
