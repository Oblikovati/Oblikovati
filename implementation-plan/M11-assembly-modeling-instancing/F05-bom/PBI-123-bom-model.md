---
milestone: M11
feature: F05
pbi: PBI-123
title: BOM views, rows, structure & quantity
status: planned
estimate: L
---

# PBI-123 — BOM views, rows, structure & quantity

**Milestone:** M11 Assembly Modeling & Instancing  ·  **Feature:** F05 Bill of Materials

## Goal

Implement the BOM derived from occurrence structure with structured/parts-only views, BOM structure per component, quantities, and item numbering.

## Scope / work

- `BOMView` structured vs parts-only.
- `BOMStructure` per definition/occurrence.
- `BOMQuantity`; item-number assignment.

## API contracts (interfaces / enums / collections)

- `BOM`,`BOMView(s)`,`BOMRow`,`BOMQuantity`,`BOMStructureEnum`

## Acceptance criteria

- The BOM reflects structure & quantities; phantom components collapse; item numbers are stable.

## Depends on

_See feature dependencies._
