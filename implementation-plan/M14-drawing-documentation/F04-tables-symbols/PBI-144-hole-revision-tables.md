---
milestone: M14
feature: F04
pbi: PBI-144
title: Hole tables, revision tables & sketched symbols
status: planned
estimate: M
---

# PBI-144 — Hole tables, revision tables & sketched symbols

**Milestone:** M14 Drawing & Documentation  ·  **Feature:** F04 Tables, Balloons & Sketched Symbols

## Goal

Implement hole tables (driven by hole features and a datum origin), revision tables/tags, general custom tables, and reusable sketched symbols/notes.

## Scope / work

- `HoleTable` from holes + origin.
- `RevisionTable`/tags.
- `CustomTable`; `SketchedSymbol` + leader/notes.

## API contracts (interfaces / enums / collections)

- `HoleTable(s)`,`RevisionTable(s)`,`CustomTable(s)`,`SketchedSymbol(s)`,`DrawingNote(s)`,`LeaderNote(s)`

## Acceptance criteria

- A hole table lists holes with X/Y from a datum; sketched symbols place with leaders.

## Depends on

_See feature dependencies._
