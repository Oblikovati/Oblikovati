---
milestone: M20
feature: F02
pbi: PBI-174
title: Sweep & loft NURBS bodies
status: planned
estimate: L
---

# PBI-174 — Sweep & loft NURBS bodies

**Milestone:** M20 Feature Completion & Geometry Parity  ·  **Feature:** F02 Swept-Surface Generation

## Goal

Sweep a profile along a path and loft between sections into real B-rep solids/surfaces.

## Scope / work

Profile-along-path sweep (translational + path frames); ruled/NURBS loft between 2+ sections; closed/open; rib as a bounded thin sweep.

## API contracts (interfaces / enums / collections)

- `SweepFeature`/`LoftFeature`/`RibFeature` real geometry; `ops.Sweep`/`ops.Loft`.

## Acceptance criteria

- A circle swept along an L-path is a validated solid
- a loft between two squares is manifold
- rib fills to the next face
- recompute on input change.

## Depends on

M07, M08

## Notes

Follows the canonical feature pattern set by Extrude (M08·PBI-092): full
`Definition → Add → Feature(+Proxy)` triangle, `/api` `wire`+`client`+`contract`,
`addin/router` handler, `.obk` round-trip, and executable tests.
