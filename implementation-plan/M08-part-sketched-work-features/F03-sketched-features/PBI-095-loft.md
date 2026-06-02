---
milestone: M08
feature: F03
pbi: PBI-095
title: Loft feature
status: planned
estimate: L
---

# PBI-095 — Loft feature

**Milestone:** M08 Part Modeling: Sketched & Work Features  ·  **Feature:** F03 Sketched Features

## Goal

Implement loft (blend through sections with rails/centerline, tangency/condition control) as the full triangle.

## Scope / work

- `LoftDefinition`; sections, `LoftRails`, `LoftSectionDimensions`.
- Tangency/condition; closed loft.

## API contracts (interfaces / enums / collections)

- `LoftFeature`,`LoftFeatures`,`LoftDefinition`,`LoftRail(s)`,`LoftSectionDimension(s)`

## Acceptance criteria

- A multi-section loft with rails blends smoothly and recomputes.

## Depends on

_See feature dependencies._
