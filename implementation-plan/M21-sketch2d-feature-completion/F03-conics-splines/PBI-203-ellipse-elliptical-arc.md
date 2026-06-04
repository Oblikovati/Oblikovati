---
milestone: M21
feature: F03
pbi: PBI-203
title: Ellipse & elliptical arc
status: planned
estimate: M
---

# PBI-203 — Ellipse & elliptical arc

**Milestone:** M21  ·  **Feature:** F03 Conics & Splines

## Goal

Expose ellipse creation and add the new `SketchEllipticalArc` entity, both with sampling
for region detection, API, and tools.

## Scope / work

- **/source:** `model/sketch` — surface existing `Ellipse`; add `EllipticalArc` type +
  collection (center, major axis dir, major/minor radii, start/end angles), backed by
  `kernel/geom` ellipse evaluation; `circularVars`-style solver hooks; sampling in
  `curvesample.go`; serialize round-trip.
- **/api:** `addEntity` kinds `ellipse`, `ellipticalArc`; `client.Sketch.AddEllipse/
  AddEllipticalArc`.
- **UI:** ellipse + elliptical-arc tools, ribbon commands, e2e.

## Acceptance criteria

- Dogfood + UI e2e create both; sampled points lie on the analytic curve (tolerance test).
- Closed ellipse forms a region; round-trip preserves both. `make ci` green.

## Depends on

PBI-202.
