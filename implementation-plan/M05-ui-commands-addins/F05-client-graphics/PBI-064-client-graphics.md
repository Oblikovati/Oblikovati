---
milestone: M05
feature: F05
pbi: PBI-064
title: Client graphics datasets, nodes & primitives
status: planned
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

## Depends on

_See feature dependencies._
