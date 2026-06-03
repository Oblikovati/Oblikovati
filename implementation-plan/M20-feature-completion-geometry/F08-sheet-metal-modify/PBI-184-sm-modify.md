---
milestone: M20
feature: F08
pbi: PBI-184
title: Cut, Rip, PunchTool, CosmeticBend, Unfold/Refold
status: planned
estimate: L
---

# PBI-184 — Cut, Rip, PunchTool, CosmeticBend, Unfold/Refold

**Milestone:** M20 Feature Completion & Geometry Parity  ·  **Feature:** F08 Sheet-Metal Modify & Cosmetic

## Goal

The SM modify features: thickness-aware cut, rip a closed wall open, place a punch, mark a cosmetic bend, and unfold/refold selected bends in the model.

## Scope / work

`CutFeature` (SM cut normal to face, optional follow-thickness); `RipFeature` (open a wall along a line/points); `PunchToolFeature` (place an iFeature punch); `CosmeticBendFeature` (annotation bend); `UnfoldFeature`/`RefoldFeature` (flatten/restore selected bends).

## API contracts (interfaces / enums / collections)

- `CutFeature(s)`/`RipFeature(s)`/`PunchToolFeature(s)`/`CosmeticBendFeature(s)`/`UnfoldFeature(s)`/`RefoldFeature(s)`.

## Acceptance criteria

- An SM cut removes material through the wall
- a rip opens a closed box
- unfold flattens one bend then refold restores it
- validated
- recompute.

## Depends on

M20·F06, M20·F01

## Notes

Follows the canonical feature pattern set by Extrude (M08·PBI-092): full
`Definition → Add → Feature(+Proxy)` triangle, `/api` `wire`+`client`+`contract`,
`addin/router` handler, `.obk` round-trip, and executable tests.
