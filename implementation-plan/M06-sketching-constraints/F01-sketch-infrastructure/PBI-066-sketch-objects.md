---
milestone: M06
feature: F01
pbi: PBI-066
title: PlanarSketch/Sketch3D containers & collections
status: planned
estimate: M
---

# PBI-066 — PlanarSketch/Sketch3D containers & collections

**Milestone:** M06 2D/3D Sketching & Constraint Solver  ·  **Feature:** F01 Sketch Infrastructure

## Goal

Implement the sketch container objects with creation on a plane/face/work-plane, the owning collections, and sketch-to-model coordinate mapping.

## Scope / work

- `Sketches.Add(planarEntity)`; `Sketches3D.Add`.
- Sketch coordinate system; `SketchToModelSpace`/`ModelToSketchSpace`.
- Edit/exit; `SketchEvents`.

## API contracts (interfaces / enums / collections)

- `PlanarSketch`,`Sketches`,`Sketch3D`,`Sketches3D`,`SketchEvents`

## Acceptance criteria

- A sketch is created on a work-plane and maps points 2D↔3D correctly.

## Depends on

_See feature dependencies._
