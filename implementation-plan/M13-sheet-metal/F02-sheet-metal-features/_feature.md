---
milestone: M13
feature: F02
name: Sheet Metal Wall & Bend Features
status: planned
---

# M13 · F02 — Sheet Metal Wall & Bend Features

The wall-creation and bending feature set that builds sheet-metal parts: base/secondary faces, flanges, contour flanges, hems, bends, folds, corner treatments, lofted flanges, and contour rolls — each the full Definition→Add→Feature triangle.

## In scope

- Face; Flange; ContourFlange; Hem; Bend; Fold.
- Corner/CornerChamfer/CornerRound.
- LoftedFlange; ContourRoll.

## Out of scope

_None._

## Key API contracts delivered

- `FaceFeature(s)`,`FlangeFeature(s)`,`ContourFlangeFeature(s)`,`HemFeature(s)`,`BendFeature(s)`,`FoldFeature(s)`,`CornerChamferFeature(s)`,`CornerRoundFeature(s)`,`LoftedFlangeFeature(s)`,`ContourRollFeature(s)`

## Depends on

F01.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-132](PBI-132-face-flange-hem.md) | Face, flange, contour flange & hem |
| [PBI-133](PBI-133-bend-fold-corner.md) | Bend, fold, corner & lofted/contour features |
