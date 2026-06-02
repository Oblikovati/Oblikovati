---
milestone: M13
feature: F03
pbi: PBI-134
title: Cut, rip, lip, punch, cosmetic bend, corner seam
status: planned
estimate: L
---

# PBI-134 — Cut, rip, lip, punch, cosmetic bend, corner seam

**Milestone:** M13 Sheet Metal  ·  **Feature:** F03 Sheet Metal Modify Features

## Goal

Implement the sheet-metal modify features, including cuts that unfold across bends and punch tools driven by iFeatures.

## Scope / work

- `CutFeature` with across-bend & normal options.
- `RipFeature`/`LipFeature`/`CornerSeam`.
- `PunchToolFeature` (iFeature punches); `CosmeticBendFeature`.

## API contracts (interfaces / enums / collections)

- `CutFeature(s)`,`RipFeature(s)`,`LipFeature(s)`,`PunchToolFeature(s)`,`CosmeticBendFeature(s)`,`CornerSeam(s)`

## Acceptance criteria

- A cut across a bend unfolds correctly; a punch tool stamps a feature.

## Depends on

_See feature dependencies._
