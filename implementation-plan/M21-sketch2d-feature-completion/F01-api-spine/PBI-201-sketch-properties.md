---
milestone: M21
feature: F01
pbi: PBI-201
title: Sketch properties (name/visible/color/linetype/lineweight/defer)
status: planned
estimate: S
---

# PBI-201 — Sketch properties

**Milestone:** M21 Sketch2D Feature Completion  ·  **Feature:** F01 API Spine & Properties

## Goal

Expose the Inventor `Sketch` scalar properties through `/api` and make them
round-trippable: Name, Visible, Color, LineType, LineWeight, DeferUpdates.

## Scope / work

- **/api:** `types/sketch_linetype.go` (`SketchLineType` enum, stable ids);
  `wire.SetSketchPropertyArgs` (sketch index + property key + value);
  `MethodSketchSetProperty`; `client.Sketch.SetProperty` + typed helpers
  (`SetName/SetVisible/SetColor/SetLineType/SetLineWeight/SetDeferUpdates`).
- **/source:** add missing fields/accessors on `sketch.Sketch` (color, lineType,
  lineWeight, deferUpdates) with serialize round-trip; router handler.

## API contracts

- `types.SketchLineType`; `wire.SetSketchPropertyArgs`; `client.Sketch.SetProperty(...)`

## Acceptance criteria

- Dogfood: set each property via `client`, re-`Get`, values match.
- Save→load round-trip preserves all six properties.
- `make ci` green.

## Depends on

PBI-200.
