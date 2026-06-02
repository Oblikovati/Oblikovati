---
milestone: M16
name: Visualization, Appearances, Styles & Presentations
status: planned
---

# M16 — Visualization, Appearances, Styles & Presentations

The visual layer: appearances/materials (physical + visual properties), styles/standards, cameras/lights/views, and presentation documents (exploded views, tweaks, storyboards/animations). This milestone is the *model-side* of visualization; it pairs with the rendering engine (see the realtime-3d-app-architecture skill) which consumes its tessellation (M07), appearances, cameras, and lights.

## Goals

- Appearance & material assets with physical and visual properties.
- A style/standard system with libraries and events.
- Cameras, lights, and views with orientation/zoom/orbit.
- Presentation documents: exploded views, tweaks, storyboards, animations.

## In scope

- `Asset`/`AssetLibrary`; `Appearance`/`Material`; physical properties.
- `StyleManager`; color/lighting styles; libraries; events.
- `Camera`; `Light(s)`; `View(s)`/`ClientViews`.
- `PresentationDocument`; exploded views; tweaks; storyboards/snapshots.

## Out of scope (handled elsewhere)

- The GPU renderer/scene graph (separate engine; see realtime-3d skill).
- Drawing rendering style (M14).

## Exit criteria

- A part shows an assigned appearance with correct color/texture and physical mass.
- The camera orbits/zooms and named views restore orientation.
- An exploded presentation animates component tweaks along trails.

## Depends on

M07, M11

## Features

| ID | Feature | PBIs | Summary |
|----|---------|:----:|---------|
| **F01** | [Appearances & Materials](F01-appearances-materials/_feature.md) | 2 | Asset-based appearances and materials with physical properties. |
| **F02** | [Styles & Standards](F02-styles-standards/_feature.md) | 1 | The style manager, color/lighting styles, libraries. |
| **F03** | [Cameras, Lights & Views](F03-cameras-lights-views/_feature.md) | 2 | Camera, lights, and named views with orbit/zoom. |
| **F04** | [Presentations & Exploded Views](F04-presentations/_feature.md) | 2 | Exploded views, tweaks, storyboards, animations. |
