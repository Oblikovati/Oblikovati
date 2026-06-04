---
milestone: M21
feature: F02
pbi: PBI-202
title: Lines/circles/arcs/points — API + tools + e2e
status: planned
estimate: M
---

# PBI-202 — Lines/circles/arcs/points

**Milestone:** M21  ·  **Feature:** F02 Curve & Point Entities

## Goal

Expose creation of the four core analytic entities through `/api` with every Inventor
overload, and ship the interactive tools to full DoD.

## Scope / work

- **/api:** `wire/sketch_entities.go` — `AddSketchEntityArgs` (sketch index, `Kind`,
  point list, radius/flags), `AddSketchEntityResult` (entity index/id, point ids);
  `MethodSketchAddEntity`; `types.SketchEntityKind` members. `client.Sketch.AddLine/
  AddCircleByCenterRadius/AddCircleByThreeTangents/AddArcByCenter/AddArcByThreePoints/
  AddArcTangent/AddPoint`.
- **/source:** add any missing model overloads on `Circles`/`Arcs` (3-tangent circle,
  tangent arc) reusing the constraint solver; `addEntity` router dispatch.
- **UI:** ribbon Create-panel commands; tools (`app/sketch_*_tool.go`) — bring the
  existing Line/Circle/Arc tools to parity, add point + arc/circle overloads; e2e
  (`app/sketch_*_tool_test.go`) driving click→commit→assert entity geometry.

## API contracts

- `wire.AddSketchEntityArgs/Result`, `MethodSketchAddEntity`
- `client.Sketch.{AddLine,AddCircleByCenterRadius,AddArcByCenter,AddPoint,...}`

## Acceptance criteria

- Dogfood: each overload creates the entity with correct geometry; enumerated by F01.
- UI e2e: line/circle/arc/point tools each commit a correct entity.
- Round-trip preserves construction flag. `make ci` green.

## Depends on

PBI-200.
