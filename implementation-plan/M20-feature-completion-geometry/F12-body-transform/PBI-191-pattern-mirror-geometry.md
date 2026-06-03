---
milestone: M20
feature: F12
pbi: PBI-191
title: Pattern & mirror real duplication
status: planned
estimate: L
---

# PBI-191 — Pattern & mirror real duplication

**Milestone:** M20 Feature Completion & Geometry Parity  ·  **Feature:** F12 Body Transform Features

## Goal

Make the existing Rectangular/Circular/SketchDriven patterns and Mirror emit real transformed copies via the new op.

## Scope / work

Wire `ops.TransformBody` into the pattern/mirror recompute so each active element is a real placed copy (booleaned into the result for join, kept separate for new-body); per-element suppression honored.

## API contracts (interfaces / enums / collections)

- `RectangularPatternFeature`/`CircularPatternFeature`/`SketchDrivenPatternFeature`/`MirrorFeature` real geometry.

## Acceptance criteria

- A 1×3 rectangular pattern of a boss yields three real placed solids at the right pitch
- mirror reflects the source across a plane
- suppressing element 2 removes only it
- recompute.

## Depends on

M07

## Notes

Follows the canonical feature pattern set by Extrude (M08·PBI-092): full
`Definition → Add → Feature(+Proxy)` triangle, `/api` `wire`+`client`+`contract`,
`addin/router` handler, `.obk` round-trip, and executable tests.
