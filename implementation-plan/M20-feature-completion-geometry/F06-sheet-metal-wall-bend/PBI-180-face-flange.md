---
milestone: M20
feature: F06
pbi: PBI-180
title: Face & Flange features
status: planned
estimate: M
---

# PBI-180 — Face & Flange features

**Milestone:** M20 Feature Completion & Geometry Parity  ·  **Feature:** F06 Sheet-Metal Wall & Bend Features

## Goal

The base `FaceFeature` (a thickened sketch wall) and `FlangeFeature` (a wall + bend off an edge).

## Scope / work

`FaceFeature` = profile thickened by rule thickness; `FlangeFeature` off a selected edge at an angle with a bend of the rule radius; flange width/extents; bend region geometry.

## API contracts (interfaces / enums / collections)

- `FaceFeature(s)`/`FlangeFeature(s)`/`*Definition`.

## Acceptance criteria

- A face creates a flat wall of rule thickness
- a 90° flange off its edge adds a wall joined by a bend arc of the rule radius
- validated solid
- recompute on angle.

## Depends on

M20·F05

## Notes

Follows the canonical feature pattern set by Extrude (M08·PBI-092): full
`Definition → Add → Feature(+Proxy)` triangle, `/api` `wire`+`client`+`contract`,
`addin/router` handler, `.obk` round-trip, and executable tests.
