---
milestone: M03
feature: F02
pbi: PBI-035
title: Documents collection & create-from-template
status: planned
estimate: M
---

# PBI-035 — Documents collection & create-from-template

**Milestone:** M03 Documents, Persistence & Identity  ·  **Feature:** F02 Documents Collection & Open/Save

## Goal

Implement the documents collection with `Add(type, template, visible)` and enumeration of visible/loaded documents.

## Scope / work

- `Add`,`Count`,`VisibleDocuments`,`LoadedCount`,indexer/ItemByName.
- Template resolution.

## API contracts (interfaces / enums / collections)

- `Documents`,`DocumentsEnumerator`

## Acceptance criteria

- A new part is created from a template and appears in the collection.
- Visible/loaded counts are correct.

## Depends on

_See feature dependencies._
