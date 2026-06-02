---
milestone: M14
feature: F02
pbi: PBI-140
title: Section, detail, auxiliary & broken views
status: planned
estimate: L
---

# PBI-140 — Section, detail, auxiliary & broken views

**Milestone:** M14 Drawing & Documentation  ·  **Feature:** F02 Drawing Views

## Goal

Implement the derived view types (section with cut hatching, detail with boundary/scale, auxiliary along a folding line, break/crop) all associative to their parent view.

## Scope / work

- Section line → section view + hatching.
- Detail boundary + scale.
- Auxiliary; break/crop.

## API contracts (interfaces / enums / collections)

- `SectionDrawingView`,`DetailDrawingView`,`AuxiliaryDrawingView`,`DrawingView` break/crop

## Acceptance criteria

- A section view shows correct cut faces with hatch; detail scales a region.

## Depends on

_See feature dependencies._
