---
milestone: M06
feature: F01
pbi: PBI-067
title: Project geometry & reference into sketch
status: planned
estimate: M
---

# PBI-067 — Project geometry & reference into sketch

**Milestone:** M06 2D/3D Sketching & Constraint Solver  ·  **Feature:** F01 Sketch Infrastructure

## Goal

Implement projecting model edges/vertices/loops and cut edges into the sketch as reference/projected geometry with associativity.

## Scope / work

- Project edge/vertex/loop; project cut edges.
- Associative update on model change.
- Include/break-link.

## API contracts (interfaces / enums / collections)

- `SketchEntity` projected variants, `Sketch.ProjectCutEdges`

## Acceptance criteria

- A projected edge updates when the source edge changes.

## Depends on

_See feature dependencies._
