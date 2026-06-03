---
milestone: M20
feature: F04
pbi: PBI-178
title: Shell & thicken
status: planned
estimate: M
---

# PBI-178 — Shell & thicken

**Milestone:** M20 Feature Completion & Geometry Parity  ·  **Feature:** F04 Local Face Operations

## Goal

Hollow a solid to a wall thickness with removed faces, and thicken a surface to a solid.

## Scope / work

Inward/outward/both shell with removed-face openings; thicken a surface body by a thickness (offset both sides + side walls); lineage.

## API contracts (interfaces / enums / collections)

- `ShellFeature`/`ThickenFeature` real geometry; `ops.Shell`/`ops.Thicken`.

## Acceptance criteria

- Shelling a box with the top removed yields a 5-wall open box of the right wall thickness, validated
- thickening a planar patch yields a slab solid
- recompute on thickness change.

## Depends on

M07, M09

## Notes

Follows the canonical feature pattern set by Extrude (M08·PBI-092): full
`Definition → Add → Feature(+Proxy)` triangle, `/api` `wire`+`client`+`contract`,
`addin/router` handler, `.obk` round-trip, and executable tests.
