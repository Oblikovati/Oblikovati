---
milestone: M10
feature: F02
pbi: PBI-111
title: Trim & extend surfaces
status: planned
estimate: M
---

# PBI-111 — Trim & extend surfaces

**Milestone:** M10 Surfacing & Freeform Modeling  ·  **Feature:** F02 Surface Editing

## Goal

Implement trimming surfaces by cutting tools and extending surface edges to boundaries/distances.

## Scope / work

- `TrimFeature` cut tool & keep-side.
- `ExtendFeature` distance/to-face, edge selection.

## API contracts (interfaces / enums / collections)

- `TrimFeature(s)`,`ExtendFeature(s)`

## Acceptance criteria

- A surface trims along a curve; an edge extends to a target.

## Depends on

_See feature dependencies._
