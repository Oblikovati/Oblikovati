---
milestone: M21
feature: F04
pbi: PBI-205
title: Rectangles, slots, polygons
status: planned
estimate: L
---

# PBI-205 — Rectangles, slots, polygons

**Milestone:** M21  ·  **Feature:** F04 Composite & Parametric Entities

## Goal

Implement the parametric multi-segment shapes Inventor exposes as single commands:
rectangle variants, the five slot constructors, and N-sided polygons — each composing
primitives + auto-constraints so they stay editable.

## Scope / work

- **/source:** `model/sketch/composite.go` — builders returning the created entity set:
  `AddRectangleThreePoint/Center`, `AddArcSlotByCenter/ByThreePoints`,
  `AddStraightSlotByCenterToCenter/ByOverall/BySlotCenter`, `AddPolygon(n, inscribed)`.
  Each emits lines/arcs + the geometric constraints (parallel/perpendicular/equal/
  tangent/coincident) that hold the shape; serialize round-trip.
- **/api:** `addEntity` kinds `rectangle`/`slot`/`polygon` with a sub-variant selector;
  `client` helpers.
- **UI:** rectangle (3-pt/center), slot (3 tools), polygon tool +
  `head/ui/sketch_polygon_dialog.go` (side count, inscribed/circumscribed); e2e.

## Acceptance criteria

- Dogfood + UI e2e: each shape lands with the right entity count and auto-constraints;
  a dimensioned slot solves to 0 DOF. Round-trip preserves the constraint set.
- `make ci` green.

## Depends on

PBI-202, F06 auto-constraints.
