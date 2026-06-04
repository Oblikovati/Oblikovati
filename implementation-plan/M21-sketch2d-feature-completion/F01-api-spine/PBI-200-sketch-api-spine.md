---
milestone: M21
feature: F01
pbi: PBI-200
title: Sketch contract/wire/client spine + enumeration + router
status: planned
estimate: M
---

# PBI-200 — Sketch contract/wire/client spine + enumeration + router

**Milestone:** M21 Sketch2D Feature Completion  ·  **Feature:** F01 API Spine & Properties

## Goal

Stand up the public-API backbone for sketches so the rich `model/sketch` is reachable
through `/api`: enumerate sketches/entities/constraints/dimensions, edit/solve a sketch,
and establish the discriminated-method scaffolding the rest of M21 extends.

## Scope / work

- **/api (Apache-2.0):**
  - `contract/sketch.go`: `Sketch`, `SketchEntity` scalar interfaces.
  - `wire/methods.go`: `MethodSketchList/Get/Edit/ExitEdit/Solve/Delete/Entities/Constraints/Dimensions`.
  - `wire/sketch.go`: `SketchInfo`, `ListSketchesResult`, `GetSketchResult`,
    `SketchEntityInfo`, `EnumerateEntitiesResult`, `ConstraintInfo`, `DimensionInfo`.
  - `client/sketch.go`: a `Sketch` group with `List/Get/Edit/ExitEdit/Solve/Delete` +
    `Entities/Constraints/Dimensions`.
- **/source (GPL):**
  - `addin/router/sketch.go`: handlers for the new methods, registered in `router.go`.
  - `var _ contract.Sketch = (*sketch.Sketch)(nil)` compile-time assertion in `model/sketch`.
  - Reference-key surfacing for sketch + entities (`GetReferenceKey`) reusing `model/identity`.

## API contracts (interfaces / enums / DTOs)

- `contract.Sketch`, `contract.SketchEntity`
- `wire.SketchInfo/SketchEntityInfo/ConstraintInfo/DimensionInfo` + the list/get DTOs
- `client.Sketch.{List,Get,Edit,ExitEdit,Solve,Delete,Entities,Constraints,Dimensions}`

## Acceptance criteria

- Dogfood: `client.Sketch().List()` returns the seeded sketch; `.Entities(i)` enumerates
  the rectangle's lines; `.Solve(i)` reports DOF; `.Edit/.ExitEdit` toggle edit state
  (`addin/router/sketch_test.go`).
- `var _ contract.Sketch` compiles; `/api` does not import `/source` (CI).
- `make ci` green; SPDX headers present.

## Depends on

M06 model, M05 router.
