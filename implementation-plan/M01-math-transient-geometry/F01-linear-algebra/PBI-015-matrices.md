---
milestone: M01
feature: F01
pbi: PBI-015
title: Matrix & Matrix2d transforms
status: planned
estimate: M
---

# PBI-015 — Matrix & Matrix2d transforms

**Milestone:** M01 Math & Transient Geometry  ·  **Feature:** F01 Linear Algebra Primitives

## Goal

Implement transformation matrices (translation/rotation/scale/mirror), composition, inversion, and point/vector transformation.

## Scope / work

- Set from axes/origin; compose; invert; determinant.
- Transform points/vectors/unit-vectors.
- Rigid vs affine distinction.

## API contracts (interfaces / enums / collections)

- `Matrix`,`Matrix2d`

## Acceptance criteria

- Round-trip transform∘inverse = identity within tolerance.
- Used as `ComponentOccurrence.Transformation` later.

## Depends on

_See feature dependencies._
