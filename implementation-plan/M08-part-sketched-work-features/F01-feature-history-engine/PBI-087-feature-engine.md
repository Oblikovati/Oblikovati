---
milestone: M08
feature: F01
pbi: PBI-087
title: Feature-history recompute engine
status: planned
estimate: XL
---

# PBI-087 — Feature-history recompute engine

**Milestone:** M08 Part Modeling: Sketched & Work Features  ·  **Feature:** F01 Feature History Engine

## Goal

Implement the engine that evaluates the ordered feature program, recomputes only the affected dependent tail in history order, and never aborts the whole rebuild on a single failure.

## Scope / work

- Ordered list; dirty propagation from inputs (params/refs).
- History-ordered re-evaluation; current-body cache advance.
- Failure isolation → feature goes sick, dependents poisoned.

## API contracts (interfaces / enums / collections)

- `PartFeatures`,`PartFeature`,`HealthStatusEnum`

## Acceptance criteria

- Editing an early feature recomputes it and its dependents only.
- A failing feature goes sick without aborting the rebuild.

## Depends on

_See feature dependencies._

## Notes

This is the realization of 'model = evaluated program'. Reuses the dirty-propagation discipline from M02-F04.
