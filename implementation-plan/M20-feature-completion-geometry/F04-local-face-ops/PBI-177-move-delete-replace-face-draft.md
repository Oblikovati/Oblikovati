---
milestone: M20
feature: F04
pbi: PBI-177
title: Move / delete / replace face & draft
status: planned
estimate: M
---

# PBI-177 — Move / delete / replace face & draft

**Milestone:** M20 Feature Completion & Geometry Parity  ·  **Feature:** F04 Local Face Operations

## Goal

Local operations that retopologize selected faces of the running solid.

## Scope / work

Move-face (translate/rotate a face, re-trim neighbours); offset-face; delete-face + heal; replace-face; face draft about a pull direction.

## API contracts (interfaces / enums / collections)

- `MoveFaceFeature`/`FaceOffsetFeature`/`DeleteFaceFeature`/`ReplaceFaceFeature`/`FaceDraftFeature` real geometry.

## Acceptance criteria

- Moving the top face of a box up grows the solid and stays manifold
- deleting a face then healing closes the body
- draft tilts the face by the angle
- recompute on input.

## Depends on

M07, M09

## Notes

Follows the canonical feature pattern set by Extrude (M08·PBI-092): full
`Definition → Add → Feature(+Proxy)` triangle, `/api` `wire`+`client`+`contract`,
`addin/router` handler, `.obk` round-trip, and executable tests.
