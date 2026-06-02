---
milestone: M01
name: Math & Transient Geometry
status: planned
---

# M01 — Math & Transient Geometry

The mathematical vocabulary of the whole system: ownerless, immutable value types (points, vectors, matrices, curves, surfaces) created by the `TransientGeometry` factory and used as arguments and results everywhere. These have no identity, no parent, and are never persisted — they are pure math, kept allocation-light because they are created in huge volume.

## Goals

- A complete 2D & 3D geometric value-type library.
- A single `TransientGeometry` factory as the one discoverable construction point.
- Curve/surface evaluators and basic geometric queries (intersection, distance, transform).

## In scope

- Points/vectors/unit-vectors/matrices (2D & 3D).
- Analytic curves (line/arc/circle/ellipse) and surfaces (plane/cyl/cone/sphere/torus).
- BSpline/NURBS curves & surfaces.
- Bounding boxes, transforms, intersections, distances.

## Out of scope (handled elsewhere)

- Persistent B-rep topology (M07).
- Sketch entities & constraints (M06).

## Exit criteria

- Any geometry needed by sketches/features can be constructed transiently and evaluated.
- All value types marshal by-value across the interop boundary.

## Depends on

M00

## Features

| ID | Feature | PBIs | Summary |
|----|---------|:----:|---------|
| **F01** | [Linear Algebra Primitives](F01-linear-algebra/_feature.md) | 2 | Points, vectors, unit vectors, and matrices in 2D and 3D. |
| **F02** | [Transient Curves](F02-transient-curves/_feature.md) | 2 | Analytic 2D/3D curves: lines, arcs, circles, ellipses. |
| **F03** | [Transient Surfaces & Splines](F03-transient-surfaces-splines/_feature.md) | 2 | Analytic surfaces plus BSpline/NURBS curves and surfaces. |
| **F04** | [Geometry Utilities & Factory](F04-geometry-utilities/_feature.md) | 3 | The TransientGeometry factory, boxes, and geometric queries. |
