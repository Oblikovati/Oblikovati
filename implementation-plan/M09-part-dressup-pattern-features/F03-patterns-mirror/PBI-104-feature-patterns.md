---
milestone: M09
feature: F03
pbi: PBI-104
title: Rectangular & circular feature patterns
status: planned
estimate: L
---

# PBI-104 — Rectangular & circular feature patterns

**Milestone:** M09 Part Modeling: Dress-up & Pattern Features  ·  **Feature:** F03 Patterns & Mirror

## Goal

Implement rectangular and circular patterns of features with parametric count/spacing/angle and per-element suppression.

## Scope / work

- Two-direction rectangular; circular about axis.
- `FeaturePatternElement` enumeration & suppress.
- Compute method (identical/adjust).

## API contracts (interfaces / enums / collections)

- `RectangularPatternFeature(s)`,`CircularPatternFeature(s)`,`FeaturePatternElement(s)`

## Acceptance criteria

- Patterns replicate the source feature; editing count updates instances; an element can be suppressed.

## Depends on

_See feature dependencies._
