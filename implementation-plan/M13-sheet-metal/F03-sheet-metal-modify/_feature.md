---
milestone: M13
feature: F03
name: Sheet Metal Modify Features
status: planned
---

# M13 · F03 — Sheet Metal Modify Features

Features that modify sheet-metal walls: cuts (normal-to-face option), rips (to open a corner), lips, punch tools (iFeature-based), cosmetic bends, and corner seams.

## In scope

- Cut (across bends).
- Rip; Lip.
- PunchTool; CosmeticBend; CornerSeam.

## Out of scope

_None._

## Key API contracts delivered

- `CutFeature(s)`,`RipFeature(s)`,`LipFeature(s)`,`PunchToolFeature(s)`,`CosmeticBendFeature(s)`,`CornerSeam(s)`

## Depends on

F02.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-134](PBI-134-cut-rip-punch.md) | Cut, rip, lip, punch, cosmetic bend, corner seam |
