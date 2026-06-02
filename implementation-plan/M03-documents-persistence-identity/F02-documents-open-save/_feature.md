---
milestone: M03
feature: F02
name: Documents Collection & Open/Save
status: planned
---

# M03 · F02 — Documents Collection & Open/Save

The `Documents` collection that owns all in-memory documents and the create-from-template, open (with options), save, and close operations, including visibility control.

## In scope

- `Documents.Add/Open/OpenWithOptions/CloseAll`.
- Templates via `FileManager.GetTemplateFile`.
- Save/SaveAs/Save copy; visible vs hidden open.

## Out of scope

_None._

## Key API contracts delivered

- `Documents`,`DocumentsEnumerator`
- template & save APIs

## Depends on

F01.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-035](PBI-035-documents-collection.md) | Documents collection & create-from-template |
| [PBI-036](PBI-036-open-save-close.md) | Open/OpenWithOptions/Save/Close lifecycle |
