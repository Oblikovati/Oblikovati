---
milestone: M11
feature: F05
name: Bill of Materials
status: planned
---

# M11 · F05 — Bill of Materials

The BOM model derived from assembly structure: views (structured/parts-only), rows with BOM structure (normal/phantom/reference/purchased), quantities, and item numbering that feeds parts lists and balloons (M14).

## In scope

- `BOM`/`BOMView`/`BOMRow`.
- `BOMStructure` (normal/phantom/reference/inseparable/purchased).
- `BOMQuantity`; item numbers; export.

## Out of scope

_None._

## Key API contracts delivered

- `BOM`,`BOMView`,`BOMViews`,`BOMRow`,`BOMRowsEnumerator`,`BOMQuantity`,`BOMStructureEnum`

## Depends on

F02.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-123](PBI-123-bom-model.md) | BOM views, rows, structure & quantity |
| [PBI-124](PBI-124-bom-export.md) | BOM export & custom columns |
