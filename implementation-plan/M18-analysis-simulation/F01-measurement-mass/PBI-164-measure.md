---
milestone: M18
feature: F01
pbi: PBI-164
title: Measurement tools
status: planned
estimate: M
---

# PBI-164 — Measurement tools

**Milestone:** M18 Analysis, Measurement & Simulation  ·  **Feature:** F01 Measurement & Mass Properties

## Goal

Implement measurement of distance/angle/area/loop length and minimum distance between entities, available interactively and via API.

## Scope / work

- Point/edge/face distance & angle.
- Area/perimeter; min-distance.
- `MeasureEvents`.

## API contracts (interfaces / enums / collections)

- `MeasureTools`,`MeasureEvents`

## Acceptance criteria

- Min-distance between two faces matches reference within tolerance.

## Depends on

_See feature dependencies._
