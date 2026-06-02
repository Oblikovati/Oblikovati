---
milestone: M16
feature: F03
name: Cameras, Lights & Views
status: planned
---

# M16 · F03 — Cameras, Lights & Views

The viewing model: cameras (position/target/up/projection), lights and lighting styles, and the views/client-views that present a document, with orientation, zoom-to-fit, and orbit operations and camera events.

## In scope

- `Camera` (eye/target/up/projection/fit).
- `Light(s)`/`LightingStyle`.
- `View(s)`/`ClientViews`; named views; orbit/zoom.
- `CameraEvents`.

## Out of scope

_None._

## Key API contracts delivered

- `Camera`,`CameraEvents`,`Light`,`Lights`,`LightingStyle`,`View`,`Views`,`ClientViews`

## Depends on

F02.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-154](PBI-154-camera-views.md) | Camera, named views & navigation |
| [PBI-155](PBI-155-lights.md) | Lights & lighting styles |
