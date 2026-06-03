---
milestone: M21
feature: F01
name: API Spine & Properties
status: done
---

# M21 · F01 — API Spine & Properties

The public-API backbone for the sketch domain: the `contract.Sketch` interfaces, the
`wire` method constants + DTOs, the `client.Sketch` typed group, and the
`addin/router` handlers — establishing the **discriminated-method** pattern (a single
`addEntity`/`addConstraint`/`addDimension` keyed on a `Kind` string, mirroring the
`WorkPlanes.Create(Kind)` slice) that every later feature extends. Plus the sketch's
own properties.

## In scope

- `sketch.list` / `sketch.get` / `sketch.edit` / `sketch.exitEdit` / `sketch.solve` /
  `sketch.delete`.
- `sketch.entities` / `sketch.constraints` / `sketch.dimensions` enumeration DTOs.
- `sketch.setProperty`: Name, Visible, Color, LineType, LineWeight, DeferUpdates.
- `contract.Sketch`, `contract.SketchEntity`; `var _` compile-time assertions.
- Reference keys (`GetReferenceKey`) for sketch + entities.

## Out of scope

- Creating entities/constraints/dimensions (F02–F07 add the `Kind` cases).

## Key API contracts delivered

- `contract.Sketch`, `contract.SketchEntity`
- `wire`: `MethodSketchList/Get/Edit/ExitEdit/Solve/Delete/SetProperty/Entities/Constraints/Dimensions`
- `types.SketchLineType`, `types.SketchEntityKind` (enum scaffolding)
- `client.Sketch` group (`List/Get/Edit/ExitEdit/Solve/Delete/SetProperty`)

## Depends on

M06 model, M05 router/commands.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-200](PBI-200-sketch-api-spine.md) | Sketch contract/wire/client spine + enumeration + router |
| [PBI-201](PBI-201-sketch-properties.md) | Sketch properties (name/visible/color/linetype/lineweight/defer) |
