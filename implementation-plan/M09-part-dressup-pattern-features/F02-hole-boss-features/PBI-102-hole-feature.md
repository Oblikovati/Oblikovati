---
milestone: M09
feature: F02
pbi: PBI-102
title: Hole feature (types, placement, tap)
status: planned
estimate: L
---

# PBI-102 — Hole feature (types, placement, tap)

**Milestone:** M09 Part Modeling: Dress-up & Pattern Features  ·  **Feature:** F02 Hole & Boss Features

## Goal

Implement holes with all placement methods and types, threaded/tapped specs, and terminations, as the full triangle.

## Scope / work

- Linear/concentric/on-point/sketch placement.
- Simple/cbore/csink/spotface; clearance vs tapped.
- `HoleTapInfo`; depth/through/to.

## API contracts (interfaces / enums / collections)

- `HoleFeature`,`HoleFeatures`,`HoleDefinition`,`HolePlacementDefinition`,`HoleTapInfo`

## Acceptance criteria

- Each placement & type builds correctly and recomputes.
- Tap data is available to hole tables (M14).

## Depends on

_See feature dependencies._
