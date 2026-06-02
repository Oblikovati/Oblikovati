---
milestone: M13
feature: F04
pbi: PBI-136
title: Flat pattern DXF/DWG export & punch reps
status: planned
estimate: M
---

# PBI-136 — Flat pattern DXF/DWG export & punch reps

**Milestone:** M13 Sheet Metal  ·  **Feature:** F04 Flat Pattern

## Goal

Implement export of the flat pattern (geometry + bend lines + punch representation) to DXF/DWG for manufacturing.

## Scope / work

- Flat geometry → DXF/DWG layers.
- Punch representation tokens.
- Export options bag.

## API contracts (interfaces / enums / collections)

- `FlatPattern` export API,`PunchRepresentation`

## Acceptance criteria

- The flat exports to DXF with bend lines and punch tags on configured layers.

## Depends on

_See feature dependencies._
