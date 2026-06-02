---
milestone: M10
feature: F03
pbi: PBI-114
title: Freeform edit operations
status: planned
estimate: L
---

# PBI-114 — Freeform edit operations

**Milestone:** M10 Surfacing & Freeform Modeling  ·  **Feature:** F03 Freeform Modeling

## Goal

Implement free-form editing (move/scale/rotate cage selections, crease, subdivide, bridge, symmetry).

## Scope / work

- Selection-based transforms.
- Crease/smooth; subdivide; bridge; mirror/symmetry.

## API contracts (interfaces / enums / collections)

- `FreeformFeature` edit API,`AliasFreeformFeature(s)`

## Acceptance criteria

- Editing the cage smoothly deforms the body; creases sharpen edges.

## Depends on

_See feature dependencies._
