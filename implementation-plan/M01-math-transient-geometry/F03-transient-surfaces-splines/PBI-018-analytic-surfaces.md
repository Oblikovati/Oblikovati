---
milestone: M01
feature: F03
pbi: PBI-018
title: Analytic surfaces (plane/cylinder/cone/sphere/torus)
status: planned
estimate: M
---

# PBI-018 — Analytic surfaces (plane/cylinder/cone/sphere/torus)

**Milestone:** M01 Math & Transient Geometry  ·  **Feature:** F03 Transient Surfaces & Splines

## Goal

Implement the analytic surface value types used as the geometry behind analytic B-rep faces.

## Scope / work

- Construction & parameterization for each.
- Point/normal/tangent evaluation.

## API contracts (interfaces / enums / collections)

- `Plane`,`Cylinder`,`Cone`,`Sphere`,`Torus`

## Acceptance criteria

- Surface point/normal match analytic reference.
- Used later as `Face.Geometry`.

## Depends on

_See feature dependencies._
