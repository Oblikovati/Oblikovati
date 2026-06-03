---
milestone: M20
feature: F02
pbi: PBI-173
title: Revolve & coil surfaces of revolution
status: planned
estimate: M
---

# PBI-173 — Revolve & coil surfaces of revolution

**Milestone:** M20 Feature Completion & Geometry Parity  ·  **Feature:** F02 Swept-Surface Generation

## Goal

Generate solids of revolution from a profile + axis so `RevolveFeature` and `CoilFeature` stop deferring.

## Scope / work

Analytic surface-of-revolution generator (full + partial angle); cap faces; helical sweep for coil (pitch/revolutions/taper); lineage on all faces.

## API contracts (interfaces / enums / collections)

- `RevolveFeature`/`CoilFeature` real geometry; `ops.Revolve`.

## Acceptance criteria

- A rectangle revolved 360° about an offset axis is a validated manifold annulus solid
- a partial angle yields start/end cap faces
- coil generates the helical body
- recompute on angle/pitch change.

## Depends on

M07, M08

## Notes

Follows the canonical feature pattern set by Extrude (M08·PBI-092): full
`Definition → Add → Feature(+Proxy)` triangle, `/api` `wire`+`client`+`contract`,
`addin/router` handler, `.obk` round-trip, and executable tests.
