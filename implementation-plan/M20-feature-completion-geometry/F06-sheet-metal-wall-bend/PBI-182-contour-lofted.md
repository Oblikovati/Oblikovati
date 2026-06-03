---
milestone: M20
feature: F06
pbi: PBI-182
title: ContourFlange, ContourRoll & LoftedFlange
status: planned
estimate: M
---

# PBI-182 — ContourFlange, ContourRoll & LoftedFlange

**Milestone:** M20 Feature Completion & Geometry Parity  ·  **Feature:** F06 Sheet-Metal Wall & Bend Features

## Goal

Sweep/loft the SM wall set: contour flange (open profile swept along edge), contour roll (revolved), lofted flange (between two profiles).

## Scope / work

ContourFlange = open profile thickened+swept along a path; ContourRoll = revolved SM wall; LoftedFlange = transition between two sketch profiles with bends.

## API contracts (interfaces / enums / collections)

- `ContourFlangeFeature(s)`/`ContourRollFeature(s)`/`LoftedFlangeFeature(s)`/`*Definition`.

## Acceptance criteria

- A contour flange sweeps its profile into a multi-bend wall
- a lofted flange transitions square→round
- validated solids
- recompute on input.

## Depends on

M20·F05

## Notes

Follows the canonical feature pattern set by Extrude (M08·PBI-092): full
`Definition → Add → Feature(+Proxy)` triangle, `/api` `wire`+`client`+`contract`,
`addin/router` handler, `.obk` round-trip, and executable tests.
