---
milestone: M21
feature: F05
pbi: PBI-209
title: Offset sketch entities
status: planned
estimate: M
---

# PBI-209 — Offset sketch entities

**Milestone:** M21  ·  **Feature:** F05 Reference & Image Entities

## Goal

Offset a sketch entity or an end-to-end connected chain by a distance or through a point,
emitting new entities + an offset constraint that keeps them parametric.

## Scope / work

- **/source:** surface/extend the existing offset in `model/sketch` —
  `OffsetUsingDistance(base, distance, side)` and `OffsetUsingPoint(base, throughPoint)`;
  emit the offset chain + `OffsetConstraint` (F06). Serialize round-trip.
- **/api:** `MethodSketchOffset`, `wire.OffsetSketchArgs`; `client.Sketch.Offset`.
- **UI:** offset tool (pick chain, drag distance / pick point), ribbon Modify command,
  e2e.

## Acceptance criteria

- Dogfood + UI e2e: offsetting a 3-segment chain by d produces a parallel chain at d with
  an offset constraint; editing d via the dimension updates it. Round-trip preserved.
- `make ci` green.

## Depends on

PBI-202, F06 offset constraint.
