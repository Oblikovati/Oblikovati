---
milestone: M13
feature: F01
name: Sheet Metal Environment & Rules
status: planned
---

# M13 · F01 — Sheet Metal Environment & Rules

The sheet-metal specialization of the part definition, the styles/rules system (material thickness, bend radius, relief, K-factor) and the unfold methods (K-factor/bend-table/equation) that govern flat development.

## In scope

- `SheetMetalComponentDefinition`.
- `SheetMetalStyles`/rules; thickness/bend-radius/relief.
- Unfold methods (K-factor/bend table/custom equation).

## Out of scope

_None._

## Key API contracts delivered

- `SheetMetalComponentDefinition`,`SheetMetalStyle(s)`,`UnfoldMethod(s)`,`BendTable`

## Depends on

M08.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-131](PBI-131-sheet-metal-rules.md) | Sheet-metal definition, styles & unfold methods |
