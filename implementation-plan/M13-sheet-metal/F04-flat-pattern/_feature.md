---
milestone: M13
feature: F04
name: Flat Pattern
status: planned
---

# M13 · F04 — Flat Pattern

The flat-pattern model that develops the folded part into a manufacturable flat, with unfold/refold features, flat extents/bend lines/punch representations, and export to DXF/DWG for cutting/laser.

## In scope

- `FlatPattern` generation.
- Unfold/Refold features.
- Bend lines/extents/punch reps.
- Flat DXF/DWG export.

## Out of scope

_None._

## Key API contracts delivered

- `FlatPattern`,`UnfoldFeature(s)`,`RefoldFeature(s)`,`Bend(s)`,`PunchRepresentation`

## Depends on

F02,F03.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-135](PBI-135-flat-pattern.md) | Flat pattern generation & unfold/refold |
| [PBI-136](PBI-136-flat-export.md) | Flat pattern DXF/DWG export & punch reps |
