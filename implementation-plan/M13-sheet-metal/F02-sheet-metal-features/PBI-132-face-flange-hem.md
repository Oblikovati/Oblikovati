---
milestone: M13
feature: F02
pbi: PBI-132
title: Face, flange, contour flange & hem
status: planned
estimate: L
---

# PBI-132 — Face, flange, contour flange & hem

**Milestone:** M13 Sheet Metal  ·  **Feature:** F02 Sheet Metal Wall & Bend Features

## Goal

Implement base/secondary faces, flanges (with bend), contour flanges (profile-driven), and hems as the full triangle, honoring the active sheet-metal rule.

## Scope / work

- `FaceFeature`; `FlangeFeature` edge/angle/height.
- `ContourFlangeFeature` profile sweep.
- `HemFeature` types.

## API contracts (interfaces / enums / collections)

- `FaceFeature(s)`,`FlangeFeature(s)`,`ContourFlangeFeature(s)`,`HemFeature(s)`

## Acceptance criteria

- A flange on an edge creates a correct bend; rule changes update geometry.

## Depends on

_See feature dependencies._
