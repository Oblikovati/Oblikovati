---
milestone: M14
feature: F01
pbi: PBI-137
title: Drawing document, sheets, borders & title blocks
status: planned
estimate: L
---

# PBI-137 — Drawing document, sheets, borders & title blocks

**Milestone:** M14 Drawing & Documentation  ·  **Feature:** F01 Drawing Document & Sheets

## Goal

Implement the drawing document, sheets (sizes/orientation), and parametric borders/title blocks with prompted property fields sourced from iProperties.

## Scope / work

- `Sheets.Add` sizes/formats.
- `BorderDefinition`/`TitleBlockDefinition` (sketch + prompts).
- Property-driven title-block fields.

## API contracts (interfaces / enums / collections)

- `DrawingDocument`,`Sheet(s)`,`Border(s)`,`TitleBlock(s)`,`*Definition(s)`

## Acceptance criteria

- A sheet with a title block shows property-driven fields that update with iProperties.

## Depends on

_See feature dependencies._
