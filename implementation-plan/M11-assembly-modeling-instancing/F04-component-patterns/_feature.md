---
milestone: M11
feature: F04
name: Component Patterns, Mirror & Substitution
status: planned
---

# M11 · F04 — Component Patterns, Mirror & Substitution

Assembly-level replication: circular/rectangular/feature-based occurrence patterns, component mirror, copy, and substitution (replace an occurrence with a simplified representation).

## In scope

- Circular/rectangular/feature-based occurrence patterns.
- MirrorComponents; CopyComponents.
- Substitute occurrences.

## Out of scope

_None._

## Key API contracts delivered

- `CircularOccurrencePattern`,`RectangularOccurrencePattern`,`FeatureBasedOccurrencePattern`,`OccurrencePatternElement`
- `MirrorComponentsDefinition`,`CopyComponentsDefinition`,`DerivedAssemblyOccurrence(s)`

## Depends on

F02.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-121](PBI-121-occurrence-patterns.md) | Component patterns (circular/rectangular/feature-based) |
| [PBI-122](PBI-122-mirror-copy-substitute.md) | Mirror, copy & substitute components |
