---
milestone: M20
feature: F07
pbi: PBI-183
title: Corner, CornerChamfer, CornerRound & CornerSeam
status: planned
estimate: M
---

# PBI-183 — Corner, CornerChamfer, CornerRound & CornerSeam

**Milestone:** M20 Feature Completion & Geometry Parity  ·  **Feature:** F07 Sheet-Metal Corner Features

## Goal

Treat the meeting of two flanges: round/chamfer the corner and apply a seam relief.

## Scope / work

`CornerFeature` (relief at a bend corner); `CornerChamferFeature`; `CornerRoundFeature`; `CornerSeamFeature` (gap/overlap/rip between adjacent walls).

## API contracts (interfaces / enums / collections)

- `CornerFeature(s)`/`CornerChamferFeature(s)`/`CornerRoundFeature(s)`/`CornerSeamFeature(s)`.

## Acceptance criteria

- Two flanges meeting at a corner get a seam with the specified gap
- a corner-round rounds the outer corner
- validated
- recompute on gap/radius.

## Depends on

M20·F06

## Notes

Follows the canonical feature pattern set by Extrude (M08·PBI-092): full
`Definition → Add → Feature(+Proxy)` triangle, `/api` `wire`+`client`+`contract`,
`addin/router` handler, `.obk` round-trip, and executable tests.
