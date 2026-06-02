---
milestone: M13
feature: F04
pbi: PBI-135
title: Flat pattern generation & unfold/refold
status: planned
estimate: L
---

# PBI-135 — Flat pattern generation & unfold/refold

**Milestone:** M13 Sheet Metal  ·  **Feature:** F04 Flat Pattern

## Goal

Implement flat-pattern development from the folded model with bend allowances, plus unfold/refold features for cut-while-flat workflows.

## Scope / work

- `FlatPattern` with bend lines & extents.
- `UnfoldFeature`/`RefoldFeature`.
- Bend-allowance per rule; A/B-side.

## API contracts (interfaces / enums / collections)

- `FlatPattern`,`UnfoldFeature(s)`,`RefoldFeature(s)`,`Bend(s)`

## Acceptance criteria

- The flat develops with correct length per K-factor; unfold/refold round-trips.

## Depends on

_See feature dependencies._
