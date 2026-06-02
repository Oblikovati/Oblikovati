---
milestone: M14
feature: F05
name: Drawing Output
status: planned
---

# M14 · F05 — Drawing Output

Producing deliverables from drawings: printing/plotting with scale/area control and export to PDF and to DWG/DXF (with model-edge layer mapping) for downstream consumers.

## In scope

- `DrawingPrintManager` print/plot.
- PDF export.
- DWG/DXF export with layer mapping.

## Out of scope

_None._

## Key API contracts delivered

- `DrawingPrintManager`,`PrintManager`
- export translators (link to M17)

## Depends on

F02.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-145](PBI-145-drawing-output.md) | Print/plot & PDF/DWG/DXF export |
