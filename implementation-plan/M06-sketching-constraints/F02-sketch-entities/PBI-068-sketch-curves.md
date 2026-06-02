---
milestone: M06
feature: F02
pbi: PBI-068
title: Sketch lines/arcs/circles/ellipses
status: planned
estimate: M
---

# PBI-068 — Sketch lines/arcs/circles/ellipses

**Milestone:** M06 2D/3D Sketching & Constraint Solver  ·  **Feature:** F02 Sketch Entities

## Goal

Implement the analytic sketch curve entities with their `Add` overloads and constrainable endpoints/center points.

## Scope / work

- `SketchLines.AddByTwoPoints` etc.; arcs/circles/ellipses.
- Endpoints/center as `SketchPoint`.
- Construction vs normal geometry.

## API contracts (interfaces / enums / collections)

- `SketchLine(s)`,`SketchArc(s)`,`SketchCircle(s)`,`SketchEllipse`,`SketchPoint(s)`

## Acceptance criteria

- Curves are created and their points participate in constraints.

## Depends on

_See feature dependencies._
