---
milestone: M21
feature: F05
pbi: PBI-210
title: Sketch image
status: planned
estimate: S
---

# PBI-210 — Sketch image

**Milestone:** M21  ·  **Feature:** F05 Reference & Image Entities

## Goal

Place a raster image on the sketch plane as a `SketchImage` entity (anchor, size,
rotation, opacity) — the underlay used for tracing.

## Scope / work

- **/source:** `model/sketch/image.go` — `SketchImage` (image ref/bytes, anchor point,
  width/height, rotation, opacity); collection; serialize round-trip (image stored in the
  `.obk` package store).
- **/api:** `addEntity` kind `image`, `wire.AddSketchImageArgs`; `client.Sketch.AddImage`.
- **UI:** insert-image tool + `head/ui/sketch_image_dialog.go` (placement/scale/opacity);
  e2e (headless: assert placement, not pixels).

## Acceptance criteria

- Dogfood + UI e2e: image places with the given anchor/size; round-trip preserves the
  reference and transform. `make ci` green.

## Depends on

PBI-202, M03 package store.
