---
milestone: M21
feature: F07
pbi: PBI-214
title: Offset/3-point-angle/ellipse/tangent/spline dimensions
status: planned
estimate: M
---

# PBI-214 — Advanced dimension kinds

**Milestone:** M21  ·  **Feature:** F07 Dimensional Constraints

## Goal

Add the dimension kinds the model lacks: offset (point-to-line / parallel distance),
three-point angle, ellipse radius, tangent distance, offset-spline distance, and
spline-fit-point.

## Scope / work

- **/source:** extend `model/sketch/dimension.go` with each new `DimKind` — measure
  function, residual, and variables; reuse the parameter binding. Serialize round-trip.
- **/api:** new `DimensionConstraintKind` members + `client.Dimension` helpers.
- **UI:** route them through the general Dimension tool (entity selection picks the kind);
  e2e for offset + three-point-angle as representatives.

## Acceptance criteria

- Dogfood + UI e2e: an offset dimension holds a point at distance d from a line; a
  three-point angle drives the included angle; each re-solves on edit. Round-trip preserved.
- `make ci` green.

## Depends on

PBI-213.
