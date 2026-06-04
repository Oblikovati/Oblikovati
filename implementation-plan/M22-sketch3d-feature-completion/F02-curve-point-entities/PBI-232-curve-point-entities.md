---
milestone: M22
feature: F02
pbi: PBI-232
title: 3D point/line/circle/arc entities + addEntity + tools
status: done (model+API; UI in F12)
estimate: M
---

# PBI-232 — 3D point/line/circle/arc entities + addEntity + tools

**Milestone:** M22  ·  **Feature:** F02 Curve & Point Entities

## Goal

Add the base 3D sketch entities and make them creatable through `/api` and an
interactive tool.

## Scope / work

- **/source kernel/model:** `model/sketch/entities_3d.go` — `Point3D` (exists),
  `Line3D`, `Circle3D`, `Arc3D`, `Bend3D`, each implementing `Entity` and exposing its
  defining `*Point3D` DOFs; geometry via `kernel/geom`. `Sketch3D.AddLine3D/AddCircle3D/
  AddArc3D/AddBend3D` factories registering ids.
- **/api:** `Sketch3DEntityKind` members; `AddSketch3DEntityArgs` (points, radius, axis,
  ccw, construction, entityRefs for bend); `client.Sketch3D` typed helpers.
- **router:** `sketch3d.addEntity` discriminated handler.
- **UI (`app/`):** a 3D line/circle/arc tool (`app/sketch3d_geometry_tools.go`) with
  ribbon buttons; pick points in model space.

## Acceptance criteria

- Unit: each entity's geometry + DOF correct; ≥98% line coverage on new kernel/model.
- Dogfood: add each kind via `client.Sketch3D`, enumerate, solve.
- Round-trip: each entity survives save→load.
- UI: ≥1 end-to-end test per tool driving command → commit → assert entity in model.
- `make ci` green.

## Depends on

PBI-230.
