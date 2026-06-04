---
milestone: M21
feature: F09
pbi: PBI-216
title: Move / rotate / copy
status: planned
estimate: M
---

# PBI-216 — Move / rotate / copy

**Milestone:** M21  ·  **Feature:** F09 Editing Operations

## Goal

Implement `MoveSketchObjects`, `RotateSketchObjects`, and `CopyEntitiesTo`/`CopyContentsTo`
— translate/rotate a selection by a vector/angle, optionally producing copies.

## Scope / work

- **/source:** `model/sketch/edit_ops.go` — `Move(sel, vector, copy)`,
  `Rotate(sel, center, angle, copy)`, `CopyEntitiesTo(sel, targetSketch)`,
  `CopyContentsTo(targetSketch)`. Re-bind/duplicate constraints among moved entities;
  serialize round-trip.
- **/api:** `MethodSketchTransform`, `wire.TransformSketchArgs` (op + selection + vector/
  center/angle + copy); `client.Sketch.Move/Rotate/Copy`.
- **UI:** move/rotate/copy tools (select → set vector/angle → commit), ribbon Modify panel,
  e2e.

## Acceptance criteria

- Dogfood + UI e2e: move translates the selection by the vector; rotate by the angle about
  the center; copy=true leaves the originals + returns copies. Round-trip preserved.
- `make ci` green.

## Depends on

PBI-202.
