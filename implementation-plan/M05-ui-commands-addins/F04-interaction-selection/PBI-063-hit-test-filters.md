---
milestone: M05
feature: F04
pbi: PBI-063
title: Hit-testing, pre-highlight & selection filters
status: planned
estimate: M
---

# PBI-063 — Hit-testing, pre-highlight & selection filters

**Milestone:** M05 Application UI, Commands, Interaction & Add-in Platform  ·  **Feature:** F04 Interaction & Selection

## Goal

Implement hit-testing against rendered geometry with pre-highlight and selection filters constraining pickable entity types.

## Scope / work

- Hit-test/pick at screen point.
- Pre-highlight under cursor.
- `SelectionFilterEnum` constraints; priority cycling.

## API contracts (interfaces / enums / collections)

- `HitTestManager`,`SelectionFilterEnum`

## Acceptance criteria

- Only filter-allowed entities highlight/pick.
- Overlapping entities cycle by priority.

## Depends on

_See feature dependencies._
