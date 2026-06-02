---
milestone: M03
feature: F04
pbi: PBI-041
title: FileManager, project paths & file locations
status: planned
estimate: M
---

# PBI-041 — FileManager, project paths & file locations

**Milestone:** M03 Documents, Persistence & Identity  ·  **Feature:** F04 Document References

## Goal

Implement file resolution via projects (search paths/libraries) and template/design-data locations.

## Scope / work

- `FileManager` resolve/relativize.
- Project (workspace/workgroup/library) search paths.
- `FileLocations`, template lookup.

## API contracts (interfaces / enums / collections)

- `FileManager`,`FileLocations`,`DesignProjectManager`,`DesignProject`

## Acceptance criteria

- A referenced file is resolved via project search paths.
- Template files are located by type.

## Depends on

_See feature dependencies._
