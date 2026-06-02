---
milestone: M16
feature: F03
pbi: PBI-154
title: Camera, named views & navigation
status: planned
estimate: M
---

# PBI-154 — Camera, named views & navigation

**Milestone:** M16 Visualization, Appearances, Styles & Presentations  ·  **Feature:** F03 Cameras, Lights & Views

## Goal

Implement the camera model with projection, fit, orbit/pan/zoom operations, named-view capture/restore, and camera events feeding the renderer.

## Scope / work

- `Camera` get/set & `Fit`/`GoHome`.
- Orbit/pan/zoom; named views.
- `Views`/`ClientViews`; `CameraEvents`.

## API contracts (interfaces / enums / collections)

- `Camera`,`Views`,`ClientViews`,`CameraEvents`

## Acceptance criteria

- The camera orbits/zooms; a named view restores exactly.

## Depends on

_See feature dependencies._
