---
milestone: M20
feature: F10
pbi: PBI-187
title: Boss, Emboss, Grill & Lip
status: planned
estimate: L
---

# PBI-187 — Boss, Emboss, Grill & Lip

**Milestone:** M20 Feature Completion & Geometry Parity  ·  **Feature:** F10 Plastic Part Features

## Goal

The additive plastic features: mounting boss, embossed/engraved profile, ventilation grill, mating lip/groove.

## Scope / work

`PlasticBossFeature` (head+thread+ribs); `EmbossFeature` (raise/engrave/emboss-from-face a profile by a depth — boolean on the running solid); `GrillFeature` (island/rib/spar/draft set in a boundary); `LipFeature` (lip+groove along an edge path).

## API contracts (interfaces / enums / collections)

- `EmbossFeature(s)`/`GrillFeature(s)`/`LipFeature(s)`/plastic `BossFeature` variants + `*Definition`.

## Acceptance criteria

- An emboss raises a profile region into a validated solid
- engrave cuts it
- a lip runs along a top edge
- recompute on depth.

## Depends on

M20·F01, M20·F03

## Notes

Follows the canonical feature pattern set by Extrude (M08·PBI-092): full
`Definition → Add → Feature(+Proxy)` triangle, `/api` `wire`+`client`+`contract`,
`addin/router` handler, `.obk` round-trip, and executable tests.
