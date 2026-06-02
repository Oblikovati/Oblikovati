---
milestone: M03
feature: F03
pbi: PBI-037
title: Structured storage container & streams
status: planned
estimate: L
---

# PBI-037 — Structured storage container & streams

**Milestone:** M03 Documents, Persistence & Identity  ·  **Feature:** F03 File Format & Storage

## Goal

Implement the on-disk compound-document container with hierarchical storages and named streams holding model data + thumbnails + metadata.

## Scope / work

- Storage/stream hierarchy.
- Atomic save (write-temp-rename).
- Thumbnail/preview stream.

## API contracts (interfaces / enums / collections)

- `File`,`tagSTATSTG`

## Acceptance criteria

- A model writes and reads back byte-identical streams.
- Interrupted save does not corrupt the prior file.

## Depends on

_See feature dependencies._
