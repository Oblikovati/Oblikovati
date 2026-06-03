---
milestone: M20
feature: F03
pbi: PBI-175
title: Rolling-ball edge fillet
status: planned
estimate: M
---

# PBI-175 — Rolling-ball edge fillet

**Milestone:** M20 Feature Completion & Geometry Parity  ·  **Feature:** F03 Fillet & Chamfer Geometry

## Goal

Replace a convex/concave edge with a constant-radius rolling-ball blend face.

## Scope / work

Constant-radius edge fillet (cylinder/torus blend faces); edge-set chains; variable radius + setback recorded; trim/rebuild adjacent faces; lineage preserved.

## API contracts (interfaces / enums / collections)

- `FilletFeature` real geometry; `ops.FilletEdges`.

## Acceptance criteria

- Filleting the 4 vertical edges of a box yields a validated manifold solid with 4 quarter-cylinder faces of the right radius
- lost edge → Sick
- recompute on radius change.

## Depends on

M07, M09

## Notes

Follows the canonical feature pattern set by Extrude (M08·PBI-092): full
`Definition → Add → Feature(+Proxy)` triangle, `/api` `wire`+`client`+`contract`,
`addin/router` handler, `.obk` round-trip, and executable tests.
