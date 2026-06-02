---
milestone: M03
feature: F01
pbi: PBI-033
title: Document base: identity, dirty, lifecycle
status: planned
estimate: M
---

# PBI-033 — Document base: identity, dirty, lifecycle

**Milestone:** M03 Documents, Persistence & Identity  ·  **Feature:** F01 Document Model & Types

## Goal

Implement the generic document with display name, full file/document name, dirty flag, open/initialized state, and lifecycle.

## Scope / work

- `DisplayName`/`FullDocumentName`/`FullFileName`.
- `Dirty`,`Open`,`Compacted`.
- Init-but-not-open (reference stub) state.

## API contracts (interfaces / enums / collections)

- `Document`,`_Document`,`DocumentTypeEnum`

## Acceptance criteria

- A document reports identity and dirty state.
- A reference stub exists without paging content.

## Depends on

_See feature dependencies._
