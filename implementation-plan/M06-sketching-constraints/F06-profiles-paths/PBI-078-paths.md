---
milestone: M06
feature: F06
pbi: PBI-078
title: Paths for sweeps & lofts
status: planned
estimate: M
---

# PBI-078 — Paths for sweeps & lofts

**Milestone:** M06 2D/3D Sketching & Constraint Solver  ·  **Feature:** F06 Profiles & Paths

## Goal

Implement path extraction (connected curve chains, 2D & 3D) used as sweep/loft rails and guides.

## Scope / work

- Connected-chain detection.
- `Path`/`Path3D`; tangency continuity.

## API contracts (interfaces / enums / collections)

- `Path`,`Path3D`

## Acceptance criteria

- A connected sketch chain yields a sweep path.

## Depends on

_See feature dependencies._
