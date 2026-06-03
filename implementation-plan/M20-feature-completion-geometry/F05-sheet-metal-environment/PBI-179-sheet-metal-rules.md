---
milestone: M20
feature: F05
pbi: PBI-179
title: Sheet-metal definition, styles & rules
status: planned
estimate: L
---

# PBI-179 — Sheet-metal definition, styles & rules

**Milestone:** M20 Feature Completion & Geometry Parity  ·  **Feature:** F05 Sheet-Metal Environment & Rules

## Goal

A `SheetMetalComponentDefinition` driven by a rule (thickness/bend radius/K-factor/unfold method) that every SM feature reads.

## Scope / work

`SheetMetalComponentDefinition` (wraps a part def); `SheetMetalStyle`/`SheetMetalRule` (Thickness/BendRadius/KFactor/relief); `UnfoldMethod` (linear/bend-table); active-rule resolution; .obk round-trip.

## API contracts (interfaces / enums / collections)

- `SheetMetalComponentDefinition`, `SheetMetalStyles`, `SheetMetalRule`, `KFactor`, `UnfoldMethod`, `BendAllowance` helper.

## Acceptance criteria

- A new SM part carries a default rule
- thickness/radius/K-factor read back
- a bend-allowance helper computes arc length from radius+angle+K
- round-trips through .obk.

## Depends on

M07, M08

## Notes

Follows the canonical feature pattern set by Extrude (M08·PBI-092): full
`Definition → Add → Feature(+Proxy)` triangle, `/api` `wire`+`client`+`contract`,
`addin/router` handler, `.obk` round-trip, and executable tests.
