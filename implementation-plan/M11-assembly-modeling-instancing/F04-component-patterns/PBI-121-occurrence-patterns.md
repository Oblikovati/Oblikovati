---
milestone: M11
feature: F04
pbi: PBI-121
title: Component patterns (circular/rectangular/feature-based)
status: planned
estimate: L
---

# PBI-121 — Component patterns (circular/rectangular/feature-based)

**Milestone:** M11 Assembly Modeling & Instancing  ·  **Feature:** F04 Component Patterns, Mirror & Substitution

## Goal

Implement occurrence patterns replicating components with parametric count/spacing and feature-driven placement, with per-element control.

## Scope / work

- Circular/rectangular pattern definitions.
- Feature-based (follow a part pattern).
- `OccurrencePatternElement` suppression.

## API contracts (interfaces / enums / collections)

- `CircularOccurrencePattern`,`RectangularOccurrencePattern`,`FeatureBasedOccurrencePattern`,`OccurrencePatternElement`

## Acceptance criteria

- Patterned occurrences track count edits; an element can be suppressed/repositioned.

## Depends on

_See feature dependencies._
