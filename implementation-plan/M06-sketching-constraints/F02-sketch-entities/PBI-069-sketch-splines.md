---
milestone: M06
feature: F02
pbi: PBI-069
title: Sketch splines & blocks
status: planned
estimate: M
---

# PBI-069 — Sketch splines & blocks

**Milestone:** M06 2D/3D Sketching & Constraint Solver  ·  **Feature:** F02 Sketch Entities

## Goal

Implement sketch splines (control/fit point) and reusable sketch blocks.

## Scope / work

- `SketchSpline` fit/control points; tangency handles.
- `SketchBlocks` definition/instances.

## API contracts (interfaces / enums / collections)

- `SketchSpline`,`SketchSplines`,`SketchBlocks`,`SketchBlockDefinition`

## Acceptance criteria

- A spline edits via its points; a block instances and updates with its definition.

## Depends on

_See feature dependencies._
