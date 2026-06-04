---
milestone: M22
feature: F01
name: API Spine & Properties (3D)
status: done (model+API; UI in F12)
---

# M22 · F01 — API Spine & Properties (3D)

The public-API backbone for the 3D-sketch domain, mirroring M21·F01 for 2D: the
`contract.Sketch3D` interface, the `wire` method constants + DTOs, the `client.Sketch3D`
typed group, and the `addin/router` handlers — re-using the **discriminated-method**
pattern (`addEntity`/`addConstraint`/`addDimension` keyed on a `Kind` string). Plus the
plumbing the rest of M22 needs: `Sketches3D` wired into `PartComponentDefinition`, the
`Sketch3D` entity registry/DOF/solve, and `SketchData3D` save→load round-trip.

## In scope

- `sketch3d.create` / `list` / `get` / `edit` / `exitEdit` / `solve` / `delete`.
- `sketch3d.entities` / `constraints` / `dimensions` enumeration DTOs.
- `sketch3d.setProperty`: Name, Visible, DimensionsVisible, OverrideColor, DeferUpdates.
- `contract.Sketch3D`; `var _` compile-time assertion against `model/sketch.Sketch3D`.
- `Sketches3D` on `PartComponentDefinition`; `Sketch3D` entity-id index + `Solve`/`DOF`.
- `SketchData3D` serialize/restore round-trip.

## Out of scope

- Creating entities/constraints/dimensions (F02–F06 add the `Kind` cases).

## Key API contracts delivered

- `contract.Sketch3D`
- `wire`: `MethodSketch3D{Create,List,Get,Edit,ExitEdit,Solve,Delete,SetProperty,
  Entities,Constraints,Dimensions}`
- `types.Sketch3DEntityKind`, `types.Geometric3DConstraintKind`,
  `types.Dimension3DConstraintKind` (enum scaffolding)
- `client.Sketch3D` group

## Depends on

M21·F01 (2D spine pattern), M06 model, M05 router/commands.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-230](PBI-230-sketch3d-api-spine.md) | Sketch3D contract/wire/client spine + enumeration + router + part wiring + serialize |
| [PBI-231](PBI-231-sketch3d-properties.md) | Sketch3D properties (name/visible/dimensionsVisible/color/defer) |
