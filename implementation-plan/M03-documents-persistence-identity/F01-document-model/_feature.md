---
milestone: M03
feature: F01
name: Document Model & Types
status: planned
---

# M03 · F01 — Document Model & Types

The `Document` object (file identity, display name, dirty, lifecycle, views) and its specializations (Part/Assembly/Drawing/Presentation), each exposing the appropriate content object, with a `DocumentTypeEnum` discriminator.

## In scope

- `Document`/`_Document` base members.
- `PartDocument`/`AssemblyDocument`/`DrawingDocument`/`PresentationDocument`.
- Document/content separation; `DocumentType`.

## Out of scope

_None._

## Key API contracts delivered

- `Document`,`_Document`,`DocumentTypeEnum`
- `PartDocument`,`AssemblyDocument`,`DrawingDocument`,`PresentationDocument`

## Depends on

M00.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-033](PBI-033-document-base.md) | Document base: identity, dirty, lifecycle |
| [PBI-034](PBI-034-document-types.md) | Document specializations & content exposure |
