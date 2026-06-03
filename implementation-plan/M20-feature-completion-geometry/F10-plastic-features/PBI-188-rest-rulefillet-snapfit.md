---
milestone: M20
feature: F10
pbi: PBI-188
title: Rest, RuleFillet & SnapFit
status: planned
estimate: M
---

# PBI-188 — Rest, RuleFillet & SnapFit

**Milestone:** M20 Feature Completion & Geometry Parity  ·  **Feature:** F10 Plastic Part Features

## Goal

Rest pad, rule-based fillet over many edges, and snap-fit (cantilever/annular) connectors.

## Scope / work

`RestFeature` (a raised landing); `RuleFilletFeature` (fillet edges selected by a rule — all-around/between-faces/feature) reusing F03; `SnapFitFeature` (cantilever hook or annular ring).

## API contracts (interfaces / enums / collections)

- `RestFeature(s)`/`RuleFilletFeature(s)`/`SnapFitFeature(s)` + `*Definition`.

## Acceptance criteria

- A rule fillet rounds every edge matching the rule
- a cantilever snap-fit adds a validated hook solid
- recompute.

## Depends on

M20·F01, M20·F03

## Notes

Follows the canonical feature pattern set by Extrude (M08·PBI-092): full
`Definition → Add → Feature(+Proxy)` triangle, `/api` `wire`+`client`+`contract`,
`addin/router` handler, `.obk` round-trip, and executable tests.
