---
milestone: M03
feature: F03
name: File Format & Storage
status: planned
---

# M03 · F03 — File Format & Storage

The persistence format: a structured storage container with named streams, the `DataIO` I/O object, save-time compaction, and version migration on open.

## In scope

- Structured storage (compound document) with streams.
- `DataIO` stream read/write.
- Compaction; migration/version upgrade.

## Out of scope

_None._

## Key API contracts delivered

- `DataIO`,`File`
- `tagSTATSTG` (storage stat)

## Depends on

F02.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-037](PBI-037-structured-storage.md) | Structured storage container & streams |
| [PBI-038](PBI-038-dataio.md) | DataIO stream I/O & attribute/data persistence |
| [PBI-039](PBI-039-compaction-migration.md) | Compaction & version migration |
