---
milestone: M12
feature: F04
name: Representations & Model States
status: planned
---

# M12 · F04 — Representations & Model States

The three assembly representation families — design-view (visibility/appearance/selection), positional (constraint-value overrides), level-of-detail (component suppression for performance) — plus model states.

## In scope

- `DesignViewRepresentation`.
- `PositionalRepresentation` overrides.
- `LevelOfDetailRepresentation`.
- Model states; events.

## Out of scope

_None._

## Key API contracts delivered

- `DesignViewRepresentation(s)`,`PositionalRepresentation(s)`,`LevelOfDetailRepresentation(s)`,`RepresentationsManager`
- `ModelStateEvents`,`RepresentationEvents`

## Depends on

M11.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-129](PBI-129-representations.md) | Design-view, positional & LOD representations |
