---
milestone: M13
feature: F02
pbi: PBI-133
title: Bend, fold, corner & lofted/contour features
status: planned
estimate: L
---

# PBI-133 — Bend, fold, corner & lofted/contour features

**Milestone:** M13 Sheet Metal  ·  **Feature:** F02 Sheet Metal Wall & Bend Features

## Goal

Implement bend, fold (bend along a sketch line), corner treatments, lofted flange, and contour roll.

## Scope / work

- `BendFeature`/`FoldFeature`.
- `CornerChamfer`/`CornerRound`.
- `LoftedFlange`/`ContourRoll`.

## API contracts (interfaces / enums / collections)

- `BendFeature(s)`,`FoldFeature(s)`,`CornerChamferFeature(s)`,`CornerRoundFeature(s)`,`LoftedFlangeFeature(s)`,`ContourRollFeature(s)`

## Acceptance criteria

- Each builds valid geometry and unfolds correctly.

## Depends on

_See feature dependencies._
