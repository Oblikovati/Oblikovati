---
milestone: M18
feature: F01
pbi: PBI-165
title: Mass & physical properties
status: planned
estimate: M
---

# PBI-165 — Mass & physical properties

**Milestone:** M18 Analysis, Measurement & Simulation  ·  **Feature:** F01 Measurement & Mass Properties

## Goal

Implement mass-properties computation from geometry + material density (volume, mass, center of mass, moments/products of inertia, principal axes).

## Scope / work

- Volume/mass from material density (M16).
- Center of mass; inertia tensor; principal axes.
- Accuracy levels; override mass.

## API contracts (interfaces / enums / collections)

- `MassProperties`,`Material`(M16)

## Acceptance criteria

- Computed mass/com/inertia match analytic reference for primitives.

## Depends on

_See feature dependencies._
