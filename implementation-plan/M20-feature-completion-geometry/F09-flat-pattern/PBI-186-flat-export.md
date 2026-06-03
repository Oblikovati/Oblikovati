---
milestone: M20
feature: F09
pbi: PBI-186
title: Flat-pattern DXF/DWG export
status: planned
estimate: M
---

# PBI-186 — Flat-pattern DXF/DWG export

**Milestone:** M20 Feature Completion & Geometry Parity  ·  **Feature:** F09 Flat Pattern

## Goal

Export the flat pattern outline + bend lines to a 2D DXF for manufacturing.

## Scope / work

DXF writer (outer/inner loops as polylines, bend lines on a layer); behind a thin `FlatExporter` interface.

## API contracts (interfaces / enums / collections)

- `FlatExporter`, `ExportFlatPatternDXF`.

## Acceptance criteria

- Exporting an unfolded part writes a DXF whose polyline closes to the flat extents and whose bend-line layer holds the bend lines
- re-import parses to the same loop.

## Depends on

M20·F06

## Notes

Follows the canonical feature pattern set by Extrude (M08·PBI-092): full
`Definition → Add → Feature(+Proxy)` triangle, `/api` `wire`+`client`+`contract`,
`addin/router` handler, `.obk` round-trip, and executable tests.
