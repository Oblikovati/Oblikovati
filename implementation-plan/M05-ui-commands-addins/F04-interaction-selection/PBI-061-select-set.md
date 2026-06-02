---
milestone: M05
feature: F04
pbi: PBI-061
title: SelectSet & selection management
status: planned
estimate: M
---

# PBI-061 — SelectSet & selection management

**Milestone:** M05 Application UI, Commands, Interaction & Add-in Platform  ·  **Feature:** F04 Interaction & Selection

## Goal

Implement the active selection set with add/remove/clear, multi-select, and selection-changed notification.

## Scope / work

- `SelectSet` CRUD; `Count`/indexer.
- Selection-changed events.
- Window/crossing selection.

## API contracts (interfaces / enums / collections)

- `SelectSet`,`SelectEvents`

## Acceptance criteria

- Picking entities populates the select set and fires events.

## Depends on

_See feature dependencies._
