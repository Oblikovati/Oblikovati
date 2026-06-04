---
milestone: M22
feature: F01
pbi: PBI-231
title: Sketch3D properties (name/visible/dimensionsVisible/color/defer)
status: planned
estimate: S
---

# PBI-231 — Sketch3D properties

**Milestone:** M22 Sketch3D Feature Completion  ·  **Feature:** F01 API Spine & Properties

## Goal

Expose the `Sketch3D` display/solve properties through `sketch3d.setProperty`, mirroring
the 2D `sketch.setProperty`.

## Scope / work

- **/api:** extend `SetSketch3DPropertyArgs.Property` domain to `"name" | "visible" |
  "dimensionsVisible" | "color" | "deferUpdates"`; the response is the updated
  `Sketch3DInfo`. `client.Sketch3D.SetProperty`.
- **/source:** `Sketch3D` gains `Visible/DimensionsVisible/OverrideColor/DeferUpdates`
  fields + getters/setters; router `sketch3d.setProperty` handler; serialize the props.

## Acceptance criteria

- Dogfood: set each property, `.Get` reflects it; deferUpdates suppresses auto-solve.
- Round-trip: properties survive save→load.
- `make ci` green; coverage targets met.

## Depends on

PBI-230.
