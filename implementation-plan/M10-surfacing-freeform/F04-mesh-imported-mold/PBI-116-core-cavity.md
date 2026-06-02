---
milestone: M10
feature: F04
pbi: PBI-116
title: Mold core/cavity tooling
status: planned
estimate: M
---

# PBI-116 — Mold core/cavity tooling

**Milestone:** M10 Surfacing & Freeform Modeling  ·  **Feature:** F04 Mesh, Imported Geometry & Mold

## Goal

Implement core/cavity feature that splits a tooling block by a part's silhouette/parting surface.

## Scope / work

- `CoreCavityFeature` parting surfaces & block split.
- Shrinkage allowance.

## API contracts (interfaces / enums / collections)

- `CoreCavityFeature(s)`

## Acceptance criteria

- A block splits into core and cavity around a part.

## Depends on

_See feature dependencies._
