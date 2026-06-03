---
milestone: M20
feature: F12
pbi: PBI-190
title: Body transform op & Move feature
status: planned
estimate: M
---

# PBI-190 — Body transform op & Move feature

**Milestone:** M20 Feature Completion & Geometry Parity  ·  **Feature:** F12 Body Transform Features

## Goal

A rigid transform (translate/rotate/reflect) of a B-rep body with stable lineage, plus the `MoveFeature`.

## Scope / work

`ops.TransformBody` (apply a 4×4 to every vertex, flip orientation on reflection, derive lineage); `MoveFeature` (free move/rotate of a body or face set by a transform).

## API contracts (interfaces / enums / collections)

- `ops.TransformBody`; `MoveFeature(s)`/`MoveDefinition`.

## Acceptance criteria

- Translating a box preserves volume/validity and shifts the range box
- reflecting it keeps it manifold with outward normals
- recompute on transform change.

## Depends on

M07

## Notes

Follows the canonical feature pattern set by Extrude (M08·PBI-092): full
`Definition → Add → Feature(+Proxy)` triangle, `/api` `wire`+`client`+`contract`,
`addin/router` handler, `.obk` round-trip, and executable tests.
