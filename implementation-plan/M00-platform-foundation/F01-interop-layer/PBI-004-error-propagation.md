---
milestone: M00
feature: F01
pbi: PBI-004
title: Error & exception propagation across the boundary
status: planned
estimate: S
---

# PBI-004 — Error & exception propagation across the boundary

**Milestone:** M00 Platform Foundation & Interop  ·  **Feature:** F01 Native/Managed Interop Layer

## Goal

Map native error codes to managed exceptions and ensure recompute/health failures are surfaced as state, not boundary crashes.

## Scope / work

- Native error → typed managed exception mapping.
- Distinguish hard errors from soft 'health' failures.
- Diagnostic context capture.

## API contracts (interfaces / enums / collections)

- (internal) error mapping
- ties to `HealthStatusEnum` (M08)

## Acceptance criteria

- A native failure raises a typed exception with context.
- Modeling 'sick' results never propagate as boundary exceptions.

## Depends on

_See feature dependencies._
