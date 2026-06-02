---
milestone: M14
feature: F03
pbi: PBI-141
title: Drawing & model dimensions (all types)
status: planned
estimate: L
---

# PBI-141 — Drawing & model dimensions (all types)

**Milestone:** M14 Drawing & Documentation  ·  **Feature:** F03 Dimensions & Annotations

## Goal

Implement standalone drawing dimensions and retrieval of model dimensions onto views, with baseline/ordinate/chain sets, all updating with the model.

## Scope / work

- Linear/angular/radius/diameter/arc-length.
- Baseline/ordinate/chain sets.
- Retrieve model dimensions; tolerance display.

## API contracts (interfaces / enums / collections)

- `DrawingDimension(s)`,`GeneralDimension`,`OrdinateDimension`,`BaselineDimensionSet(s)`

## Acceptance criteria

- A dimension on a view updates when the model size changes.

## Depends on

_See feature dependencies._
