---
milestone: M09
feature: F03
name: Patterns & Mirror
status: planned
---

# M09 · F03 — Patterns & Mirror

Feature patterns that replicate source features with per-element control and the mirror feature, all parametric in count/spacing and supporting suppression of individual elements.

## In scope

- Rectangular & circular feature patterns.
- Sketch-driven pattern.
- Mirror feature.
- `FeaturePatternElement` suppression.

## Out of scope

_None._

## Key API contracts delivered

- `RectangularPatternFeature(s)`,`CircularPatternFeature(s)`,`SketchDrivenPatternFeature(s)`,`MirrorFeature(s)`
- `FeaturePatternElement(s)`,`FeatureBasedPatternElement`

## Depends on

M08.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-104](PBI-104-feature-patterns.md) | Rectangular & circular feature patterns |
| [PBI-105](PBI-105-sketch-driven-mirror.md) | Sketch-driven pattern & mirror |
