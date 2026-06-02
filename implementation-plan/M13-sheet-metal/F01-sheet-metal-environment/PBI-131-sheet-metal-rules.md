---
milestone: M13
feature: F01
pbi: PBI-131
title: Sheet-metal definition, styles & unfold methods
status: planned
estimate: L
---

# PBI-131 — Sheet-metal definition, styles & unfold methods

**Milestone:** M13 Sheet Metal  ·  **Feature:** F01 Sheet Metal Environment & Rules

## Goal

Implement the sheet-metal component definition with the styles/rules driving thickness, bend radius, relief, and the unfold methods controlling bend allowance.

## Scope / work

- `SheetMetalComponentDefinition`.
- `SheetMetalStyle` (thickness/radius/relief/gap).
- `UnfoldMethod` (K-factor/bend-table/equation).

## API contracts (interfaces / enums / collections)

- `SheetMetalComponentDefinition`,`SheetMetalStyle(s)`,`UnfoldMethod(s)`,`BendTable`

## Acceptance criteria

- The active rule governs feature defaults; changing K-factor changes flat extents.

## Depends on

_See feature dependencies._
