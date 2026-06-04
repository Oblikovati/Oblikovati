---
milestone: M21
feature: F04
pbi: PBI-206
title: Sketch fillet & chamfer
status: planned
estimate: M
---

# PBI-206 — Sketch fillet & chamfer

**Milestone:** M21  ·  **Feature:** F04 Composite & Parametric Entities

## Goal

Round/bevel the corner between two adjacent sketch curves: trim both to the blend, insert
a tangent arc (fillet) or line (chamfer), and add the constraints that keep it parametric.

## Scope / work

- **/source:** `model/sketch/corner_blend.go` — `AddFillet(c1, c2, radius)` solves the
  tangent-arc center, trims/extends both curves to the tangent points, inserts the arc,
  adds tangent + coincident constraints; `AddChamfer(c1, c2, dist1, dist2|angle)` inserts
  the bevel line. Reuse the rolling-ball math style from `kernel/ops` fillet where useful;
  serialize round-trip.
- **/api:** `addEntity` kinds `fillet`/`chamfer`; `client.Sketch.AddFillet/AddChamfer`.
- **UI:** fillet + chamfer tools (pick two curves, set radius/distance), ribbon Modify
  panel, `head/ui/sketch_fillet_dialog.go`; e2e.

## Acceptance criteria

- Dogfood + UI e2e: filleting a rectangle corner inserts a tangent arc of the requested
  radius (tangency residual ≈ 0); chamfer inserts the bevel line. Round-trip preserved.
- `make ci` green.

## Depends on

PBI-205, F06 (tangent/coincident constraints).
