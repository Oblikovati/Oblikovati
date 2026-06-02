---
milestone: M14
name: Drawing & Documentation
status: planned
---

# M14 — Drawing & Documentation

The 2D documentation environment: drawing documents with sheets/borders/title blocks/standards, associative drawing views (base/projected/section/detail/etc.) generated from 3D models, dimensions and annotations (GD&T), tables (parts list/balloons/hole/revision), sketched symbols, and output (print/PDF/DWG/DXF). Views and dimensions reference model topology via reference keys (M03), so they update when the model changes.

## Goals

- Drawing documents with sheets, formats, borders, title blocks, and standards.
- Associative drawing views generated and updated from 3D models.
- Dimensions, annotations, and GD&T symbols on annotation planes.
- Parts lists, balloons, hole tables, and revision tables.
- Output: print/plot and PDF/DWG/DXF export.

## In scope

- `DrawingDocument`; sheets/formats/borders/title blocks; standards/styles.
- Drawing views (base/projected/section/detail/auxiliary/overlay/draft).
- Dimensions & annotations; centerlines; GD&T; datum frames.
- Parts lists/balloons; hole/revision/general tables.
- Sketched symbols; print; PDF/DWG/DXF export.

## Out of scope (handled elsewhere)

- 3D model creation (M06-M13).
- BOM data model (M11) — drawing consumes it.

## Exit criteria

- A base+projected view of a part renders correct hidden/visible edges.
- A model dimension placed on a view updates when the model changes.
- A parts list + balloons reflect the assembly BOM.
- The drawing exports to PDF and DWG.

## Depends on

M07, M11

## Features

| ID | Feature | PBIs | Summary |
|----|---------|:----:|---------|
| **F01** | [Drawing Document & Sheets](F01-drawing-sheets/_feature.md) | 2 | Drawing documents, sheets, borders, title blocks, standards. |
| **F02** | [Drawing Views](F02-drawing-views/_feature.md) | 2 | Base/projected/section/detail/auxiliary views from 3D. |
| **F03** | [Dimensions & Annotations](F03-dimensions-annotations/_feature.md) | 2 | Model/drawing dimensions, GD&T, centerlines, datum frames. |
| **F04** | [Tables, Balloons & Sketched Symbols](F04-tables-symbols/_feature.md) | 2 | Parts lists, balloons, hole/revision tables, sketched symbols. |
| **F05** | [Drawing Output](F05-drawing-output/_feature.md) | 1 | Print/plot and PDF/DWG/DXF export. |
