---
milestone: M08
feature: F03
pbi: PBI-093
title: Revolve feature
status: planned
estimate: M
---

# PBI-093 — Revolve feature

**Milestone:** M08 Part Modeling: Sketched & Work Features  ·  **Feature:** F03 Sketched Features

## Goal

Implement revolve (profile about an axis) with full/angle extents and operations as the full triangle.

## Scope / work

- `RevolveDefinition` (profile/axis/angle/operation).
- Full vs angular; two-directional.
- Proxy & recompute.

## API contracts (interfaces / enums / collections)

- `RevolveFeature`,`RevolveFeatures`,`RevolveDefinition`

## Acceptance criteria

- Full and partial revolves build valid solids and recompute.

## Depends on

_See feature dependencies._
