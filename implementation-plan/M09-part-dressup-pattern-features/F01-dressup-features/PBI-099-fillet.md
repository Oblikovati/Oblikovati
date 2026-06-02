---
milestone: M09
feature: F01
pbi: PBI-099
title: Fillet feature (constant/variable/setback)
status: planned
estimate: L
---

# PBI-099 — Fillet feature (constant/variable/setback)

**Milestone:** M09 Part Modeling: Dress-up & Pattern Features  ·  **Feature:** F01 Dress-up Features

## Goal

Implement fillet with constant and variable radius, edge/face/full-round sets, and setbacks, as the full triangle over edge selections.

## Scope / work

- `FilletDefinition` edge sets & radii.
- Variable-radius points; setbacks; full-round.
- Reference-keyed edge inputs; recompute.

## API contracts (interfaces / enums / collections)

- `FilletFeature`,`FilletFeatures`,`FilletDefinition`,`FilletConstantRadiusEdgeSet`,`FilletVariableRadiusEdgeSet`

## Acceptance criteria

- Constant & variable fillets build and recompute when edges move.
- Lost-edge → feature sick.

## Depends on

_See feature dependencies._
