---
milestone: M09
feature: F01
pbi: PBI-100
title: Chamfer feature
status: planned
estimate: M
---

# PBI-100 — Chamfer feature

**Milestone:** M09 Part Modeling: Dress-up & Pattern Features  ·  **Feature:** F01 Dress-up Features

## Goal

Implement chamfer (distance, distance-angle, two-distance) over edge selections.

## Scope / work

- Three chamfer modes.
- Edge sets; setback at corners.

## API contracts (interfaces / enums / collections)

- `ChamferFeature`,`ChamferFeatures`,`ChamferDefinition`

## Acceptance criteria

- All chamfer modes build valid geometry and recompute.

## Depends on

_See feature dependencies._
