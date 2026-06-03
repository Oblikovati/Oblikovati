---
milestone: M21
feature: F09
pbi: PBI-217
title: Trim / extend / split
status: planned
estimate: M
---

# PBI-217 — Trim / extend / split

**Milestone:** M21  ·  **Feature:** F09 Editing Operations

## Goal

The interactive curve-editing verbs: trim a curve back to its nearest intersections, extend
it to the nearest boundary, and split it at a point into two entities.

## Scope / work

- **/source:** `model/sketch/edit_ops.go` — `Trim(curve, atPoint)` (remove the segment
  between bounding intersections containing the pick), `Extend(curve, atEnd)` (lengthen to
  the nearest other curve), `Split(curve, atPoint)` (two curves sharing a coincident point).
  Reuse intersection helpers in `model/sketch/detection.go`; serialize round-trip.
- **/api:** `MethodSketchTrim/Extend/Split`, arg DTOs; `client.Sketch.Trim/Extend/Split`.
- **UI:** trim/extend/split tools (hover shows the affected segment, click commits), ribbon
  Modify panel, e2e.

## Acceptance criteria

- Dogfood + UI e2e: trimming a crossed line removes the picked segment; extend reaches the
  boundary; split yields two coincident curves. Round-trip preserved. `make ci` green.

## Depends on

PBI-202.
