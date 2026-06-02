---
milestone: M09
feature: F04
pbi: PBI-108
title: Thicken/offset & direct-edit
status: planned
estimate: M
---

# PBI-108 — Thicken/offset & direct-edit

**Milestone:** M09 Part Modeling: Dress-up & Pattern Features  ·  **Feature:** F04 Modify & Direct-Edit Features

## Goal

Implement thicken/offset (surface→solid or face offset) and the consolidated direct-edit feature for push/pull editing.

## Scope / work

- `ThickenFeature` (surface/solid).
- `DirectEditFeature` move/size/scale/rotate/delete.

## API contracts (interfaces / enums / collections)

- `ThickenFeature(s)`,`DirectEditFeature(s)`

## Acceptance criteria

- Thicken converts a surface to a solid; direct-edit push/pull modifies faces.

## Depends on

_See feature dependencies._
