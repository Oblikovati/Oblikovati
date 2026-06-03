---
milestone: M20
feature: F01
pbi: PBI-172
title: Cut / Split / Combine real geometry
status: planned
estimate: M
---

# PBI-172 — Cut / Split / Combine real geometry

**Milestone:** M20 Feature Completion & Geometry Parity  ·  **Feature:** F01 Intersecting Booleans

## Goal

With PBI-171 landed, complete the features that are pure booleans: `CombineFeature`
(already Phase-A), `SplitFeature` (split a body/face by a tool), and a new
`CutFeature` (subtract a tool body) — all emitting validated geometry.

## Scope / work

- `CombineFeature`: route intersecting cases through the new boolean.
- `SplitFeature`: split the running solid by a work plane / surface / sketch into
  the kept piece(s); split-faces mode for face splitting.
- `CutFeature(s)`/`CutDefinition`: subtract a tool body (full triangle + Add).
- Health: tool that misses the body → Warning, not crash.

## API contracts (interfaces / enums / collections)

- `CutFeature`,`CutFeatures`,`CutDefinition`,`CutFeatureProxy`; `SplitFeature` geometry.

## Acceptance criteria

- Splitting a box by its mid-plane yields the requested half as a validated solid.
- A cut tool overlapping a block removes exactly the overlap volume.
- Combine of two intersecting blocks (join) is manifold with the union volume.
- All recompute when a driving parameter changes; serialize round-trips.

## Depends on

PBI-171.

## Notes

Follows the canonical feature pattern set by Extrude (M08·PBI-092).
