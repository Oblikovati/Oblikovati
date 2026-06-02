---
milestone: M01
feature: F04
pbi: PBI-020
title: TransientGeometry factory (single construction point)
status: planned
estimate: M
---

# PBI-020 — TransientGeometry factory (single construction point)

**Milestone:** M01 Math & Transient Geometry  ·  **Feature:** F04 Geometry Utilities & Factory

## Goal

Expose every transient value type through one factory with consistent Create* methods and `PointTolerance`.

## Scope / work

- `CreatePoint/Vector/Matrix/Line/Arc/...` full surface.
- Allocation discipline (pooling-friendly).

## API contracts (interfaces / enums / collections)

- `TransientGeometry`

## Acceptance criteria

- Every value type is constructible via the factory.
- High-volume creation stays allocation-light.

## Depends on

_See feature dependencies._
