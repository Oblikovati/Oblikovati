---
milestone: M01
feature: F01
pbi: PBI-014
title: Point/Vector/UnitVector value types (2D & 3D)
status: planned
estimate: M
---

# PBI-014 — Point/Vector/UnitVector value types (2D & 3D)

**Milestone:** M01 Math & Transient Geometry  ·  **Feature:** F01 Linear Algebra Primitives

## Goal

Implement immutable position and direction value types with vector arithmetic, dot/cross, length, normalization, and conversions.

## Scope / work

- 2D & 3D variants.
- Arithmetic, dot/cross, angle-between, normalize.
- Conversions point↔vector↔unitvector.

## API contracts (interfaces / enums / collections)

- `Point`,`Point2d`,`Vector`,`Vector2d`,`UnitVector`,`UnitVector2d`

## Acceptance criteria

- All operations match double-precision reference within tolerance.
- By-value marshaling verified.

## Depends on

_See feature dependencies._
