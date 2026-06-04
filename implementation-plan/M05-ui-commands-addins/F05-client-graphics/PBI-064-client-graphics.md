---
milestone: M05
feature: F05
pbi: PBI-064
title: Client graphics datasets, nodes & primitives
status: in-progress
estimate: L
---

# PBI-064 — Client graphics datasets, nodes & primitives

**Milestone:** M05 Application UI, Commands, Interaction & Add-in Platform  ·  **Feature:** F05 Client Graphics

## Goal

Implement the client-graphics object model for drawing add-in-owned geometry (lines, curves, surfaces, text, markers) into the view, persisted with the document where desired.

## Scope / work

- `ClientGraphics`/`ClientGraphicsCollection` ownership.
- `GraphicsDataSets`/`GraphicsNode` hierarchy.
- Coordinate/transform & color/visibility.

## API contracts (interfaces / enums / collections)

- `ClientGraphics`,`ClientGraphicsCollection`,`GraphicsDataSets`,`GraphicsNode`

## Acceptance criteria

- An add-in draws persistent overlay geometry that saves/reloads.

## Delivered

Declarative bulk-group model (ADR-0018): one `clientGraphics.set` submits/replaces a
named group of nodes → primitives, geometry shipped as flat arrays — the right shape for
large simulation-result meshes over the wire.

- API contract (`Oblikovati.API`, Apache-2.0): `types/graphics.go` enums,
  `wire/graphics.go` DTOs (+ method constants), `contract/graphics.go` scalar view,
  `client/graphics.go` typed `Graphics()` group with `AddMesh/AddHeatmap/AddLines/
  AddPoints/AddLabel` helpers.
- Host store + builder (`model/clientgraphics/`): per-lane `Store`, decode/validate,
  `ColorMapper` (scalar→color heatmaps), point-glyph expansion, `Build(cam)` →
  `renderer.DrawItem` + world-anchored `Label`s.
- Renderer: per-vertex color (`renderer.DrawItem.Colors` + `head/viewport/flatten.go`) —
  the FEA heatmap enabler — and an `OnTop` flag.
- Router (`addin/router/graphics.go`): `clientGraphics.set/list/delete/setVisible`.
- Render integration: `app/render.go` `RenderFrame` draws the graphics; the windowed
  head draws geometry + projected text labels (`head/ui/client_graphics_overlay.go`).

**Remaining for "saves/reloads":** persistence of `saveWithDocument` groups into the
`.obk` YAML is deferred to **PBI-066** (the simulation-overlay use case is transient).
True on-top rendering (depth-disabled pass) landed in **PBI-067**.

## Depends on

_See feature dependencies._
