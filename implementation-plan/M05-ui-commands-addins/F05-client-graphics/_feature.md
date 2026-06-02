---
milestone: M05
feature: F05
name: Client Graphics
status: planned
---

# M05 · F05 — Client Graphics

The client-graphics system add-ins and commands use to draw transient/overlay geometry (previews, manipulators, decorations) in the 3D view, organized into graphics datasets/nodes with their own coordinate handling.

## In scope

- `ClientGraphics`/`ClientGraphicsCollection`.
- Graphics datasets/nodes; line/curve/surface/text primitives.
- Preview/overlay vs persistent client graphics; `InteractionGraphics`.

## Out of scope

_None._

## Key API contracts delivered

- `ClientGraphics`,`ClientGraphicsCollection`,`GraphicsDataSets`,`GraphicsNode`
- `InteractionGraphics`,`ClientGraphicsControlDefinition`

## Depends on

F04.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-064](PBI-064-client-graphics.md) | Client graphics datasets, nodes & primitives |
| [PBI-065](PBI-065-interaction-graphics.md) | Interaction (preview) graphics |
