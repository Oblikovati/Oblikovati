---
milestone: M14
feature: F02
name: Drawing Views
status: planned
---

# M14 · F02 — Drawing Views

Associative 2D views generated from 3D parts/assemblies: base and projected views, section/detail/auxiliary/overlay/break/crop views, with correct hidden-line computation, rendering style, and view-to-model edge associativity (drawing curves carry model reference keys).

## In scope

- Base/projected views.
- Section/detail/auxiliary/overlay/break/crop.
- Hidden-line/shaded rendering style; scale/alignment.
- `DrawingCurve`↔model edge associativity.

## Out of scope

_None._

## Key API contracts delivered

- `DrawingView`,`DrawingViews`,`DrawingCurve(s)`,`DrawingCurveSegment(s)`
- `SectionDrawingView`,`DetailDrawingView`,`AuxiliaryDrawingView`,`ProjectedDrawingView`,`DrawingViewEvents`

## Depends on

F01,M07,M11.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-139](PBI-139-base-projected-views.md) | Base & projected views with hidden-line |
| [PBI-140](PBI-140-section-detail-views.md) | Section, detail, auxiliary & broken views |
