---
milestone: M14
feature: F01
name: Drawing Document & Sheets
status: planned
---

# M14 · F01 — Drawing Document & Sheets

The drawing container and page model: sheets with sizes/formats, borders and title blocks (parametric, prompt-driven), and the drafting standards/styles that govern dimension/text/line appearance.

## In scope

- `DrawingDocument`; `Sheet`/`Sheets`/`SheetFormats`.
- `Border`/`TitleBlock` definitions (prompted fields).
- `DrawingStandards`/styles managers.

## Out of scope

_None._

## Key API contracts delivered

- `DrawingDocument`,`Sheet`,`Sheets`,`Border`,`BorderDefinition(s)`,`TitleBlock`,`TitleBlockDefinition(s)`
- `DrawingStylesManager`,`DrawingStandardStyle`,`DimensionStyle`,`DrawingSettings`

## Depends on

M03.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-137](PBI-137-drawing-sheets.md) | Drawing document, sheets, borders & title blocks |
| [PBI-138](PBI-138-drafting-standards.md) | Drafting standards & style managers |
