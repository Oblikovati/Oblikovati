---
milestone: M20
feature: F06
pbi: PBI-181
title: Bend, Fold & Hem features
status: planned
estimate: M
---

# PBI-181 — Bend, Fold & Hem features

**Milestone:** M20 Feature Completion & Geometry Parity  ·  **Feature:** F06 Sheet-Metal Wall & Bend Features

## Goal

`BendFeature` (bend along a line), `FoldFeature` (fold about a sketch line), `HemFeature` (folded edge).

## Scope / work

Bend across a face along a bend line; fold a wall about a line by an angle; hem (single/teardrop/rolled/double) at an edge; all consume the rule.

## API contracts (interfaces / enums / collections)

- `BendFeature(s)`/`FoldFeature(s)`/`HemFeature(s)`/`*Definition`.

## Acceptance criteria

- Folding a wall 90° about a line yields two walls + a bend
- a single hem folds the edge back the hem gap
- validated
- recompute on angle/radius.

## Depends on

M20·F05

## Notes

Follows the canonical feature pattern set by Extrude (M08·PBI-092): full
`Definition → Add → Feature(+Proxy)` triangle, `/api` `wire`+`client`+`contract`,
`addin/router` handler, `.obk` round-trip, and executable tests.
