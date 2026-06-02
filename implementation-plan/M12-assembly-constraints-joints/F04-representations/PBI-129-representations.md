---
milestone: M12
feature: F04
pbi: PBI-129
title: Design-view, positional & LOD representations
status: planned
estimate: L
---

# PBI-129 — Design-view, positional & LOD representations

**Milestone:** M12 Assembly: Constraints, Joints, Motion & Representations  ·  **Feature:** F04 Representations & Model States

## Goal

Implement the three representation families with capture/activate and override semantics, plus model states.

## Scope / work

- Design-view: visibility/appearance/section/camera.
- Positional: constraint/joint value overrides; flexible.
- LOD: suppression sets for performance.
- Model states.

## API contracts (interfaces / enums / collections)

- `DesignViewRepresentation(s)`,`PositionalRepresentation(s)`,`LevelOfDetailRepresentation(s)`,`RepresentationsManager`

## Acceptance criteria

- Switching representations changes visibility/position/suppression accordingly.

## Depends on

_See feature dependencies._
