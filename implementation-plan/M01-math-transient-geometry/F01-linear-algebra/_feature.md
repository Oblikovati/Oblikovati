---
milestone: M01
feature: F01
name: Linear Algebra Primitives
status: planned
---

# M01 · F01 — Linear Algebra Primitives

The foundational value types for position, direction, and transformation in both 2D (sketch) and 3D (model) spaces, with the arithmetic and transform operations the kernel and features rely on.

## In scope

- `Point`/`Point2d`, `Vector`/`Vector2d`, `UnitVector`/`UnitVector2d`.
- `Matrix`/`Matrix2d` with compose/invert/transform.
- Coordinate-system operations.

## Out of scope

_None._

## Key API contracts delivered

- `Point`,`Point2d`,`Vector`,`Vector2d`,`UnitVector`,`UnitVector2d`
- `Matrix`,`Matrix2d`

## Depends on

M00.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-014](PBI-014-points-vectors.md) | Point/Vector/UnitVector value types (2D & 3D) |
| [PBI-015](PBI-015-matrices.md) | Matrix & Matrix2d transforms |
