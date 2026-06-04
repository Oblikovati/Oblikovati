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
| [PBI-066](PBI-066-client-graphics-persistence.md) | Persist saveWithDocument client graphics in .obk |
| [PBI-067](PBI-067-graphics-on-top-pass.md) | Native depth-disabled pass for on-top graphics |

## Status

PBI-065 + PBI-067 done; PBI-064 in progress (in-session object model + render + on-top
pass delivered; only `.obk` persistence (PBI-066) remains). The model is **declarative
bulk groups** over the wire (one `clientGraphics.set` per group) rather than Inventor's
chatty mutable object model — the right shape for large simulation-result overlays.
Surface/Curve-from-B-rep-body primitives (`SurfaceGraphics`/`CurveGraphics`) are out of
scope for now.
