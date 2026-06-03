---
milestone: M20
feature: F09
pbi: PBI-185
title: Flat-pattern unfold solver
status: planned
estimate: M
---

# PBI-185 — Flat-pattern unfold solver

**Milestone:** M20 Feature Completion & Geometry Parity  ·  **Feature:** F09 Flat Pattern

## Goal

Compute the flat pattern: unfold every bend by its allowance into a single planar sheet with bend lines.

## Scope / work

Bend graph traversal from a base face; per-bend allowance (K-factor/bend-table) flattening; `FlatPattern` model (extents, bend lines, punch centers); kept in sync with the folded model.

## API contracts (interfaces / enums / collections)

- `FlatPattern`, `FlatPatternFeatures`, `BendLine`, bend-allowance computation.

## Acceptance criteria

- An L-shaped two-wall part unfolds to one rectangle whose length = wall1 + allowance + wall2
- bend line recorded at the right offset
- recompute when a wall length changes.

## Depends on

M20·F06

## Notes

Follows the canonical feature pattern set by Extrude (M08·PBI-092): full
`Definition → Add → Feature(+Proxy)` triangle, `/api` `wire`+`client`+`contract`,
`addin/router` handler, `.obk` round-trip, and executable tests.
