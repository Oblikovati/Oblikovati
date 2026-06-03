---
milestone: M21
name: Sketch2D Feature Completion (2D Parity)
status: planned
---

# M21 — Sketch2D Feature Completion (2D Parity)

Bring the **2D planar sketch** domain to full Autodesk Inventor `Sketch` parity
(**excluding DWG**), across all three layers: the public **API** (`/api`
contract/wire/client), the **kernel/model** that implements it (`model/sketch`,
reusing `kernel/geom`), and the **UI tooling** (`app/` commands+tools, `head/ui`
property windows) under the project Definition of Done.

The model layer is already rich (entities, ~17 geometric constraints, 5 dimension
kinds, a Newton/LM solver with DOF, profiles/regions, projection, offset, serialize
round-trip), but it is **almost entirely unexposed** through `/api` (only
`sketch.create` + `sketch.rectangle`) and only **partially driveable** in the UI.
This milestone closes that gap and fills the remaining model/kernel holes so a user
can fully model any real 2D profile interactively, with every capability reachable
through the typed `api/client` and covered by tests.

## Goals

- Every Inventor `Sketch` entity, constraint, dimension, and operation (ex-DWG)
  reachable through `/api` and through a ribbon command + interactive tool.
- A fully-constrained sketch solves deterministically to 0 DOF and reports
  `ConstraintStatus`.
- Save→load round-trip equality for every new entity/constraint/pattern.

## In scope

- Planar sketch entities to parity: lines, circles, arcs (all overloads), points,
  ellipse, elliptical arc, splines (interpolation/control/fixed/offset), equation
  curves, rectangles, slots, polygons, sketch fillet/chamfer, fill regions, text.
- Reference geometry: project/include, offset, sketch images.
- Full geometric + dimensional constraint sets; `ConstraintStatus`, DOF, defer/solve.
- Editing operations: move/rotate/copy/trim/extend/split/mirror/delete.
- Sketch rectangular & circular patterns.
- Profiles/regions/paths exposed through `/api`.

## Out of scope (handled elsewhere)

- 3D sketch (`Sketch3D`): helical/intersection/on-face/project-to-surface curves.
- DWG/drawing-sketch geometry and AutoCAD interop.
- Turning a profile into a solid (M08); annotation use on drawing sheets (M14).

## Exit criteria

- Each feature is drivable in the app (ribbon → tool/dialog → committed geometry)
  with an end-to-end test, AND callable through `api/client` (dogfood test).
- `model/sketch` covers every parity entity/constraint/op with unit + round-trip tests.
- A representative fully-constrained sketch (rectangle + slot + fillet, dimensioned)
  solves to 0 DOF and survives save→load.

## Depends on

M06 (sketch foundation), M02 (parameters/expressions), M05 (commands/UI), M01
(`kernel/geom` curves), M08 (profile consumers, for the profiles API).

## Features

| ID | Feature | PBIs | Summary |
|----|---------|:----:|---------|
| **F01** | [API Spine & Properties](F01-api-spine/_feature.md) | 2 | `contract.Sketch` + wire/client + router; list/get/edit/solve/setProperty; entity/constraint enumeration. The discriminated-method pattern. |
| **F02** | [Curve & Point Entities](F02-curve-point-entities/_feature.md) | 1 | Line, circle, arc (all overloads), point — full API + tools. |
| **F03** | [Conics & Splines](F03-conics-splines/_feature.md) | 2 | Ellipse, elliptical arc, splines (interpolation/control/fixed/offset), equation curve. |
| **F04** | [Composite & Parametric Entities](F04-composite-entities/_feature.md) | 3 | Rectangle variants, slots, polygon, sketch fillet/chamfer, fill region, text. |
| **F05** | [Reference & Image Entities](F05-reference-image/_feature.md) | 3 | Project/include, offset, sketch image. |
| **F06** | [Geometric Constraints](F06-geometric-constraints/_feature.md) | 2 | Full geometric set incl. ground, offset, align, pattern; list/show/delete. |
| **F07** | [Dimensional Constraints](F07-dimensional-constraints/_feature.md) | 2 | Full dimension set, driving/driven, limits, edit/drive. |
| **F08** | [Constraint Status & DOF](F08-constraint-status/_feature.md) | 1 | `ConstraintStatus`, DOF coloring, defer/solve, auto-dimension. |
| **F09** | [Editing Operations](F09-editing-operations/_feature.md) | 3 | Move/rotate/copy, trim/extend/split, mirror, delete. |
| **F10** | [Sketch Patterns](F10-sketch-patterns/_feature.md) | 2 | Rectangular + circular sketch patterns; pattern constraint. |
| **F11** | [Profiles & Regions API](F11-profiles-regions/_feature.md) | 1 | Profile/Profiles/ProfilePath + region detection over `/api`. |
