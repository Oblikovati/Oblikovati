---
milestone: M17
feature: F02
pbi: PBI-159
title: STEP & IGES import/export
status: planned
estimate: XL
---

# PBI-159 — STEP & IGES import/export

**Milestone:** M17 Interoperability & Translation  ·  **Feature:** F02 Neutral CAD Formats

## Goal

Implement STEP and IGES translators that import B-rep solids/surfaces (healed to valid bodies, brought in as base features) and export the model with unit/structure fidelity.

## Scope / work

- STEP AP203/214/242 read/write.
- IGES read/write.
- Heal on import (M07); assembly structure; unit mapping.

## API contracts (interfaces / enums / collections)

- STEP/IGES translators,`NonParametricBaseFeature`

## Acceptance criteria

- A STEP solid imports as a valid healed body and re-exports without degradation.

## Depends on

_See feature dependencies._
