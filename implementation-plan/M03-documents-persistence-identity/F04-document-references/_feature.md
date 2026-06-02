---
milestone: M03
feature: F04
name: Document References
status: planned
---

# M03 · F04 — Document References

The reference graph between documents (a part referenced by many assemblies/drawings), descriptors for native references, and the `FileManager`/project-path resolution that locates referenced files.

## In scope

- Referenced/referencing/all-referenced enumerations.
- `DocumentDescriptor` reference records.
- `FileManager`, project (`.ipj`-style) path resolution, `FileLocations`.

## Out of scope

_None._

## Key API contracts delivered

- `Document.ReferencedDocuments`/`ReferencingDocuments`/`AllReferencedDocuments`
- `DocumentDescriptor`,`DocumentDescriptorsEnumerator`
- `FileManager`,`FileLocations`,`DesignProjectManager`

## Depends on

F01,F02.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-040](PBI-040-reference-graph.md) | Document reference graph & descriptors |
| [PBI-041](PBI-041-filemanager-projects.md) | FileManager, project paths & file locations |
