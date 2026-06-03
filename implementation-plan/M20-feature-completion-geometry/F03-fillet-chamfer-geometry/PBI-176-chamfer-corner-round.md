---
milestone: M20
feature: F03
pbi: PBI-176
title: Chamfer & corner-round
status: planned
estimate: M
---

# PBI-176 — Chamfer & corner-round

**Milestone:** M20 Feature Completion & Geometry Parity  ·  **Feature:** F03 Fillet & Chamfer Geometry

## Goal

Flat chamfer faces (distance / distance-angle / two-distance) and corner-round.

## Scope / work

Edge chamfer planar faces; the three chamfer input modes; corner-round at vertices; lineage.

## API contracts (interfaces / enums / collections)

- `ChamferFeature` real geometry; `ops.ChamferEdges`.

## Acceptance criteria

- Chamfering a box edge yields a validated solid with one planar chamfer face at the right setback
- two-distance asymmetric verified
- recompute on distance change.

## Depends on

M07, M09

## Notes

Follows the canonical feature pattern set by Extrude (M08·PBI-092): full
`Definition → Add → Feature(+Proxy)` triangle, `/api` `wire`+`client`+`contract`,
`addin/router` handler, `.obk` round-trip, and executable tests.
