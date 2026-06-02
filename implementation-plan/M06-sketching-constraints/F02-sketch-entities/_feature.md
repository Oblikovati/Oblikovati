---
milestone: M06
feature: F02
name: Sketch Entities
status: planned
---

# M06 · F02 — Sketch Entities

The drawable sketch geometry with sketch-specific behavior (endpoints as constrainable points, on-curve points), produced by the sketch's typed entity collections.

## In scope

- `SketchLine`/`SketchArc`/`SketchCircle`/`SketchEllipse`/`SketchSpline`/`SketchPoint`.
- Sketch entity collections & `Add` overloads.
- Control/fit points; sketch blocks.

## Out of scope

_None._

## Key API contracts delivered

- `SketchLine`,`SketchLines`,`SketchArc`,`SketchArcs`,`SketchCircle`,`SketchCircles`,`SketchEllipse`,`SketchSpline`,`SketchPoint`,`SketchPoints`
- `SketchEntity`,`SketchEntities`,`SketchBlocks`

## Depends on

F01.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-068](PBI-068-sketch-curves.md) | Sketch lines/arcs/circles/ellipses |
| [PBI-069](PBI-069-sketch-splines.md) | Sketch splines & blocks |
