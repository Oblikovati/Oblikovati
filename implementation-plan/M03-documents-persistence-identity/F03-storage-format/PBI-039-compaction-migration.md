---
milestone: M03
feature: F03
pbi: PBI-039
title: Compaction & version migration
status: planned
estimate: M
---

# PBI-039 — Compaction & version migration

**Milestone:** M03 Documents, Persistence & Identity  ·  **Feature:** F03 File Format & Storage

## Goal

Implement save-time compaction and on-open migration of older document versions.

## Scope / work

- Compaction pass on save.
- Version stamp + migration pipeline.
- `OnMigrateDocument` hook (M04).

## API contracts (interfaces / enums / collections)

- `Document.Compacted`, migration pipeline

## Acceptance criteria

- Compaction reduces file size without data loss.
- An older-version file opens and migrates.

## Depends on

_See feature dependencies._
