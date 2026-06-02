---
milestone: M11
feature: F02
pbi: PBI-119
title: Transform, ground, suppress & nesting
status: planned
estimate: M
---

# PBI-119 — Transform, ground, suppress & nesting

**Milestone:** M11 Assembly Modeling & Instancing  ·  **Feature:** F02 Component Occurrences

## Goal

Implement per-occurrence placement transform and state (grounded/suppressed/adaptive) plus the nesting model (paths, sub-occurrences, parent).

## Scope / work

- `Transformation` get/set; `SetTransformWithoutConstraints`.
- `Grounded`/`Suppressed`/`Adaptive`.
- `OccurrencePath`/`SubOccurrences`/`ParentOccurrence`.

## API contracts (interfaces / enums / collections)

- `ComponentOccurrence.Transformation/Grounded/Suppressed`,`OccurrencePath`,`SubOccurrences`

## Acceptance criteria

- An occurrence moves via its transform; a subassembly's nested instances are addressable by path.

## Depends on

_See feature dependencies._
