---
milestone: M21
feature: F07
pbi: PBI-213
title: Expose linear/angular/radial dimensions
status: planned
estimate: M
---

# PBI-213 — Expose linear/angular/radial dimensions

**Milestone:** M21  ·  **Feature:** F07 Dimensional Constraints

## Goal

Make the existing dimension kinds (distance, angle, radius, diameter, arc-length) creatable
and editable through `/api` and the general Dimension tool, including driving/driven and
limits.

## Scope / work

- **/api:** `types.DimensionConstraintKind`; `wire.AddDimensionArgs` (sketch + Kind + refs
  + expression), `wire.DriveDimensionArgs` (value/driven/limits);
  `MethodSketchAddDimension/DriveDimension`; `client.Sketch.Dimension` group.
- **/source:** `addin/router/sketch_dimensions.go` mapping Kind → `DimensionConstraints.
  AddXxx`; drive + set-driven + set-limits handlers.
- **UI:** the general Dimension tool (pick entities → infer kind → edit value box) +
  `head/ui/sketch_dimension_dialog.go` for the value/limits; e2e.

## Acceptance criteria

- Dogfood + UI e2e: a distance dimension drives two points apart; editing it re-solves;
  driven flag flips; limits clamp the drive. Round-trip preserves all. `make ci` green.

## Depends on

PBI-200, M02 parameters.
