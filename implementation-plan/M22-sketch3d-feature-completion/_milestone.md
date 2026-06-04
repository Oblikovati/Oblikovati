---
milestone: M22
name: Sketch3D Feature Completion (3D Sketch Parity)
status: in-progress
---

# M22 — Sketch3D Feature Completion (3D Sketch Parity)

Bring the **3D sketch** domain to full Autodesk Inventor `Sketch3D` parity, across all
three layers: the public **API** (`/api` contract/wire/client), the **kernel/model**
that implements it (`model/sketch`, `kernel/geom`), and the **UI tooling** (`app/`
commands+tools, `head/ui` property windows) under the project Definition of Done.

Where M21 closed the 2D planar sketch, this milestone closes the non-planar one: lines,
arcs, circles, conics and splines that live freely in model space; helical curves; the
full 3D geometric + dimensional constraint sets; profiles/paths for sweep/loft
consumers; and the surface-derived curves (intersection, on-face, project-to-surface,
silhouette, offset) — the last of which requires **new kernel surface-intersection
machinery** built here (per the milestone scope decision: full scope, build all).

The starting point is thin: the model has a skeleton `Sketch3D`/`Sketches3D` (no entity
`Add*` methods, not wired into `PartComponentDefinition`), a dimension-agnostic Newton
solver that already handles 3D, and a few 3D primitives (`Point3D` + 5 constraints +
`AddDistance`). There is **zero** Sketch3D surface in `/api`, the router, or the UI.
`kernel/geom` already has the 3D curve primitives (`Line`, `Arc3d`, `Circle`,
`EllipseFull`, `EllipticalArc`, `BSplineCurve`) and the analytic surfaces
(`Plane`/`Cylinder`/`Cone`/`Sphere`/`Torus`/`BSplineSurface`), but **no helix** and
**no surface↔curve / surface↔surface intersection**.

## Goals

- Every Inventor `Sketch3D` entity, curve, constraint, dimension, and operation
  reachable through `/api` and through a ribbon command + interactive tool.
- A fully-constrained 3D sketch solves deterministically to 0 DOF and reports
  `ConstraintStatus`.
- Save→load round-trip equality for every new entity/constraint.
- New kernel surface-intersection algorithms, property/metamorphic tested.

## In scope

- 3D entities to parity: point, line, circle, arc/bend, ellipse, elliptical arc,
  splines (interpolation/control/fixed), equation curve.
- Helical curves (all four definition modes) + helical constraint.
- Surface-derived curves: intersection, on-face, project-to-surface, silhouette,
  offset (`OffsetCurve3`).
- Full 3D geometric + dimensional constraint sets; `ConstraintStatus`, DOF, defer/solve.
- Editing operations: move/rotate/copy/delete; `Include` (project part geometry).
- `Profile3D`/`Profiles3D` + open-path detection over `/api`.
- 3D sketch UI environment, ribbon command, `Sketch3DSettings` (AutoBendRadius).

## Out of scope (handled elsewhere)

- 2D planar sketch (`Sketch`) — M21.
- Consuming a 3D path/profile into a swept/lofted solid — M08/M10 feature engine.
- DWG/drawing-sketch geometry — M14.

## Exit criteria

- Each feature is drivable in the app (ribbon → tool/dialog → committed geometry) with
  an end-to-end test, AND callable through `api/client` (dogfood test).
- `model/sketch` + `kernel/geom` 3D additions covered by unit + round-trip tests;
  every new kernel public API action ≥98% line coverage; overall ≥80%.
- A representative 3D sketch (helix + on-face curve + dimensioned frame) solves to
  0 DOF and survives save→load.

## Depends on

M21 (sketch spine + discriminated-method pattern), M06 (sketch foundation), M02
(parameters/expressions), M05 (commands/UI), M01/M07 (`kernel/geom` curves+surfaces).

## Features

| ID | Feature | PBIs | Summary |
|----|---------|:----:|---------|
| **F01** | [API Spine & Properties](F01-api-spine/_feature.md) | 2 | `contract.Sketch3D` + wire/client + router; create/list/get/edit/solve/setProperty; entity/constraint/dimension enumeration; `Sketches3D` wired into the part; `SketchData3D` round-trip. |
| **F02** | [Curve & Point Entities](F02-curve-point-entities/_feature.md) | 1 | Point, line, circle, arc/bend (all overloads) — kernel + API + tools. |
| **F03** | [Conics & Splines](F03-conics-splines/_feature.md) | 2 | Ellipse, elliptical arc, splines (interpolation/control/fixed), equation curve. |
| **F04** | [Helical Curves](F04-helical-curves/_feature.md) | 1 | `Helix3d` kernel primitive + helical curve (4 modes) + helical constraint. |
| **F05** | [Geometric Constraints](F05-geometric-constraints/_feature.md) | 2 | Full 3D geometric set incl. parallel-to-axis/plane, ground, smooth, bend; list/show/delete. |
| **F06** | [Dimensional Constraints](F06-dimensional-constraints/_feature.md) | 1 | Distance/line-length/radius/point-plane/two-line-angle/spline-length; driving/driven. |
| **F07** | [Constraint Status & DOF](F07-constraint-status/_feature.md) | 1 | `ConstraintStatus`, DOF, defer/solve, DimensionsVisible. |
| **F08** | [Editing & Reference Ops](F08-editing-ops/_feature.md) | 2 | Move/rotate/copy, delete, `Include` (project part geometry), reference keys. |
| **F09** | [Profiles & Paths API](F09-profiles-paths/_feature.md) | 1 | `Profile3D`/`Profiles3D` + open-path detection over `/api`. |
| **F10** | [Surface-Intersection Kernel](F10-surface-intersection-kernel/_feature.md) | 2 | Surface↔curve, surface↔surface intersection, point projection, silhouette. |
| **F11** | [Surface-Derived Curves](F11-surface-derived-curves/_feature.md) | 2 | Intersection/on-face/project-to-surface/silhouette/offset curves + API + tools. |
| **F12** | [UI Environment & Settings](F12-ui-environment/_feature.md) | 1 | 3D Sketch ribbon + environment + `Sketch3DSettings`; e2e UI tests. |
