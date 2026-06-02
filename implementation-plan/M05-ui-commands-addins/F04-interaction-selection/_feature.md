---
milestone: M05
feature: F04
name: Interaction & Selection
status: planned
---

# M05 · F04 — Interaction & Selection

The user-input pipeline that turns clicks into model selections: the selection set, interaction events (mouse/keyboard/select), hit-testing against geometry, and selection filters that constrain what can be picked.

## In scope

- `SelectSet` selection management.
- `InteractionEvents`, `MouseEvents`, `SelectEvents`.
- Hit-test/pick; pre-highlight.
- `SelectionFilterEnum` filters.

## Out of scope

_None._

## Key API contracts delivered

- `SelectSet`,`InteractionEvents`,`MouseEvents`,`SelectEvents`,`KeyboardEvents`
- `SelectionFilterEnum`,`HitTestManager`,`InteractionManager`

## Depends on

F02.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-061](PBI-061-select-set.md) | SelectSet & selection management |
| [PBI-062](PBI-062-interaction-events.md) | Interaction & mouse/keyboard event pipeline |
| [PBI-063](PBI-063-hit-test-filters.md) | Hit-testing, pre-highlight & selection filters |
