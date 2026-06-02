---
milestone: M03
name: Documents, Persistence & Identity
status: planned
---

# M03 — Documents, Persistence & Identity

The document/file layer and the single hardest kernel problem — persistent topological identity. Documents are the file/lifecycle/reference unit, cleanly split from the modeling content they hold. This milestone delivers the document model and types, open/save, structured storage, the document reference graph, the `ReferenceKeyManager` (so topology survives recompute and reload), and the attribute/property metadata side-channel.

## Goals

- A `Document` model split from its content, with all four document types.
- Open/save/close with templates, options, and structured on-disk storage.
- The document reference graph (referenced/referencing/all-referenced).
- Reference keys: stable, serializable topology identity surviving recompute and reload.
- Extensible attribute & property metadata persisted with the model.

## In scope

- `Document`+specializations; `Documents`; open/save/close.
- Structured storage, streams, `DataIO`, compaction, migration.
- Reference graph & `FileManager`/project paths.
- `ReferenceKeyManager`, key contexts.
- `AttributeSet`/`Attribute`, `PropertySets`/iProperties.

## Out of scope (handled elsewhere)

- Document content (ComponentDefinition) modeling — M07+.
- Transactions/undo — M04.

## Exit criteria

- A document can be created from template, saved, closed, reopened, with references resolved.
- A reference key bound to a face survives save→close→reopen→rebind.
- Attributes attached to entities persist across save/reload.

## Depends on

M00, M02

## Features

| ID | Feature | PBIs | Summary |
|----|---------|:----:|---------|
| **F01** | [Document Model & Types](F01-document-model/_feature.md) | 2 | The Document base, document types, and the document/content split. |
| **F02** | [Documents Collection & Open/Save](F02-documents-open-save/_feature.md) | 2 | The in-memory document collection and open/save/close lifecycle. |
| **F03** | [File Format & Storage](F03-storage-format/_feature.md) | 3 | Structured on-disk storage, streams, DataIO, compaction, migration. |
| **F04** | [Document References](F04-document-references/_feature.md) | 2 | The cross-document reference graph and file/project resolution. |
| **F05** | [Persistent Identity (Reference Keys)](F05-reference-keys/_feature.md) | 2 | Stable, serializable topology identity surviving recompute & reload. |
| **F06** | [Attributes & Metadata](F06-attributes-metadata/_feature.md) | 2 | Extensible per-entity attributes and document iProperties. |
