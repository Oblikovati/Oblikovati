---
milestone: M14
feature: F04
name: Tables, Balloons & Sketched Symbols
status: planned
---

# M14 · F04 — Tables, Balloons & Sketched Symbols

Drawing tables and symbols: parts lists driven by the assembly BOM (M11), balloons referencing list items, hole tables driven by hole features (M09), revision tables, general tables, and reusable sketched symbols and notes.

## In scope

- `PartsList`/`Balloon(s)`; balloon styles.
- `HoleTable` from hole features.
- `RevisionTable`; general tables; `DrawingBOM`.
- Sketched symbols; `DrawingNote(s)`/leader notes.

## Out of scope

_None._

## Key API contracts delivered

- `PartsList`,`PartsLists`,`Balloon(s)`,`BalloonStyle`,`HoleTable(s)`,`RevisionTable(s)`,`CustomTable(s)`,`DrawingBOM(s)`
- `SketchedSymbol(s)`,`SketchedSymbolDefinition(s)`,`DrawingNote(s)`,`LeaderNote(s)`

## Depends on

F03,M11.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-143](PBI-143-parts-list-balloons.md) | Parts list & balloons from BOM |
| [PBI-144](PBI-144-hole-revision-tables.md) | Hole tables, revision tables & sketched symbols |
