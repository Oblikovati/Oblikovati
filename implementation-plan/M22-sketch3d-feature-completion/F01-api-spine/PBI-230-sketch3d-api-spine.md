---
milestone: M22
feature: F01
pbi: PBI-230
title: Sketch3D contract/wire/client spine + enumeration + router + part wiring + serialize
status: done (model+API; UI in F12)
estimate: M
---

# PBI-230 — Sketch3D contract/wire/client spine + enumeration + router + part wiring + serialize

**Milestone:** M22 Sketch3D Feature Completion  ·  **Feature:** F01 API Spine & Properties

## Goal

Stand up the public-API backbone for 3D sketches so `model/sketch.Sketch3D` is reachable
through `/api`: create/enumerate/edit/solve/delete a 3D sketch, enumerate its
entities/constraints/dimensions, and establish the discriminated-method scaffolding the
rest of M22 extends. Wire `Sketches3D` into the part and round-trip it through `.obk`.

## Scope / work

- **/api (Apache-2.0):**
  - `types/sketch3d_kind.go`: `Sketch3DEntityKind`, `Geometric3DConstraintKind`,
    `Dimension3DConstraintKind` enum scaffolding (members filled by F02–F06).
  - `contract/sketch3d.go`: `Sketch3D` scalar interface (`Name/Visible/EntityCount/
    DegreesOfFreedom`).
  - `wire/methods.go`: `MethodSketch3DCreate/List/Get/Edit/ExitEdit/Solve/Delete/
    SetProperty/Entities/Constraints/Dimensions`.
  - `wire/sketch3d_spine.go`: `Sketch3DArgs`, `Sketch3DInfo`, `ListSketches3DResult`,
    `Sketch3DEntityInfo`, `EnumerateEntities3DResult`, `Constraint3DInfo`,
    `Dimension3DInfo`, `SolveSketch3DResult`, `EditSketch3DResult`,
    `SetSketch3DPropertyArgs`.
  - `client/sketch3d.go`: a `Sketch3D` group with `Create/List/Get/Edit/ExitEdit/Solve/
    Delete/Entities/Constraints/Dimensions`.
- **/source (GPL):**
  - `model/sketch/sketch_variants.go` (or new `sketch3d.go`): give `Sketch3D` an entity
    registry (`byID`), `DegreesOfFreedom()`, `Solve()` (reuse `solveSketch`), and the
    constraint/dimension collections (`GeometricConstraints3D`, `DimensionConstraints3D`).
  - `model/compdef/part.go`: `sketches3D *sketch.Sketches3D` field + `Sketches3D()`.
  - `model/compdef/serialize.go` + `model/sketch/serialize*.go`: `SketchData3D` DTO,
    serialize/restore.
  - `addin/router/sketch3d_spine.go` + registration in `router.go`.
  - `var _ contract.Sketch3D = (*sketch.Sketch3D)(nil)`.

## Acceptance criteria

- Dogfood: `client.Sketch3D().Create()` then `.List()` returns it; `.Get(i)` reports
  name/visible/entityCount/DOF; `.Edit/.ExitEdit` toggle; `.Solve(i)` reports DOF
  (`addin/router/sketch3d_test.go`).
- `var _ contract.Sketch3D` compiles; `/api` does not import `/source` (CI).
- `SketchData3D` round-trips: a 3D sketch with N entities save→load equal.
- `make ci` green; SPDX headers present; coverage targets met on new files.

## Depends on

M21·F01 (2D spine), M06 model, M05 router.
