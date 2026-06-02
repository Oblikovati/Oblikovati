---
milestone: M01
feature: F04
pbi: PBI-021
title: Bounding boxes (Box, Box2d, OrientedBox)
status: planned
estimate: S
---

# PBI-021 — Bounding boxes (Box, Box2d, OrientedBox)

**Milestone:** M01 Math & Transient Geometry  ·  **Feature:** F04 Geometry Utilities & Factory

## Goal

Implement axis-aligned and oriented bounding boxes used by range queries across the model.

## Scope / work

- Min/max corners; extend/contains/intersect.
- Oriented box from axes.

## API contracts (interfaces / enums / collections)

- `Box`,`Box2d`,`OrientedBox`

## Acceptance criteria

- `RangeBox`/`OrientedMinimumRangeBox` consumers work later.

## Depends on

_See feature dependencies._
