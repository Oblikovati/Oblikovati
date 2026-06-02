---
milestone: M03
feature: F04
pbi: PBI-040
title: Document reference graph & descriptors
status: planned
estimate: M
---

# PBI-040 — Document reference graph & descriptors

**Milestone:** M03 Documents, Persistence & Identity  ·  **Feature:** F04 Document References

## Goal

Implement the in-memory reference graph and the descriptors that record native document references for lazy resolution.

## Scope / work

- Referenced/referencing/all-referenced queries.
- `DocumentDescriptor` (path, needs-update, reference key).
- Lazy load of referenced docs.

## API contracts (interfaces / enums / collections)

- `DocumentDescriptor`,`DocumentDescriptorsEnumerator`,`DocumentsEnumerator`

## Acceptance criteria

- An assembly reports its parts; a part reports its referencing assemblies.
- Broken references are flagged, not fatal.

## Depends on

_See feature dependencies._
