---
milestone: M08
feature: F03
pbi: PBI-094
title: Sweep feature
status: planned
estimate: L
---

# PBI-094 — Sweep feature

**Milestone:** M08 Part Modeling: Sketched & Work Features  ·  **Feature:** F03 Sketched Features

## Goal

Implement sweep (profile along a path, optional guide rail/surface, twist) as the full triangle.

## Scope / work

- `SweepDefinition` (profile/path/orientation/taper/twist).
- Guide rail/surface options.

## API contracts (interfaces / enums / collections)

- `SweepFeature`,`SweepFeatures`,`SweepDefinition`

## Acceptance criteria

- A profile sweeps along a 2D/3D path with correct orientation.

## Depends on

_See feature dependencies._
