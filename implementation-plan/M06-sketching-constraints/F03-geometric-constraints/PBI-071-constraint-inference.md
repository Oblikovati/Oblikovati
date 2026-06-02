---
milestone: M06
feature: F03
pbi: PBI-071
title: Constraint inference during sketching
status: planned
estimate: M
---

# PBI-071 — Constraint inference during sketching

**Milestone:** M06 2D/3D Sketching & Constraint Solver  ·  **Feature:** F03 Geometric Constraints

## Goal

Implement live inference of likely constraints (coincident/tangent/parallel/perpendicular/horizontal/vertical) as the user sketches, with glyphs.

## Scope / work

- Inference engine + priorities.
- Glyph display via interaction graphics.
- Apply-on-commit.

## API contracts (interfaces / enums / collections)

- inference over `GeometricConstraints`,`InteractionGraphics`

## Acceptance criteria

- Drawing near a horizontal infers and applies a horizontal constraint.

## Depends on

_See feature dependencies._
