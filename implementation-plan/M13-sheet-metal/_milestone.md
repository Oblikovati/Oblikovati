---
milestone: M13
name: Sheet Metal
status: planned
---

# M13 — Sheet Metal

The sheet-metal environment: a specialized part component-definition driven by sheet-metal rules (thickness, bend radius, K-factor, unfold methods), the wall/bend/corner feature set, modify features, and the flat-pattern model with unfold/refold and DXF/DWG export of the flat.

## Goals

- A sheet-metal definition with styles/rules and unfold methods.
- The wall/bend/corner feature set (face, flange, hem, fold, contour features).
- Modify features (cut, rip, punch, corner seam).
- Flat-pattern generation with unfold/refold and flat export.

## In scope

- `SheetMetalComponentDefinition`; styles/rules; K-factor; unfold methods.
- Face/Flange/ContourFlange/Hem/Bend/Fold/Corner*/LoftedFlange/ContourRoll.
- Cut/Rip/Lip/PunchTool/CosmeticBend/CornerSeam.
- `FlatPattern`; Unfold/Refold; flat export.

## Out of scope (handled elsewhere)

- Drawing of sheet-metal flat (M14).
- Generic solid features (M08/M09 reused).

## Exit criteria

- A flange added to a face creates a correct bend per the active rule.
- The flat pattern unfolds with correct bend allowances.
- The flat exports to DXF for manufacturing.

## Depends on

M08, M09

## Features

| ID | Feature | PBIs | Summary |
|----|---------|:----:|---------|
| **F01** | [Sheet Metal Environment & Rules](F01-sheet-metal-environment/_feature.md) | 1 | Sheet-metal definition, styles, rules, unfold methods. |
| **F02** | [Sheet Metal Wall & Bend Features](F02-sheet-metal-features/_feature.md) | 2 | Face, flange, contour flange, hem, bend, fold, corner. |
| **F03** | [Sheet Metal Modify Features](F03-sheet-metal-modify/_feature.md) | 1 | Cut, rip, lip, punch tool, cosmetic bend, corner seam. |
| **F04** | [Flat Pattern](F04-flat-pattern/_feature.md) | 2 | Unfold/refold, flat extents, and DXF/DWG export. |
