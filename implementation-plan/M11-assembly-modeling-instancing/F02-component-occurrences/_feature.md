---
milestone: M11
feature: F02
name: Component Occurrences
status: planned
---

# M11 · F02 — Component Occurrences

The occurrence (instance) object pairing a shared definition with a placement transform and per-instance state, nested through occurrence paths and sub-occurrences — the flyweight at the heart of assemblies.

## In scope

- Place/copy occurrence; `Definition` link.
- `Transformation`; `Grounded`/`Suppressed`/`Adaptive`.
- `OccurrencePath`/`SubOccurrences`/`ParentOccurrence`.

## Out of scope

_None._

## Key API contracts delivered

- `ComponentOccurrence`,`ComponentOccurrences`,`ComponentOccurrencesEnumerator`
- `Matrix`(M01)

## Depends on

F01.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-118](PBI-118-place-occurrence.md) | Place/copy occurrences & shared definitions |
| [PBI-119](PBI-119-occurrence-state.md) | Transform, ground, suppress & nesting |
