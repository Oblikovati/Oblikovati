---
milestone: M08
feature: F01
name: Feature History Engine
status: planned
---

# M08 · F01 — Feature History Engine

The recompute engine that owns the ordered feature list, evaluates the dependent tail in history order on change, carries per-feature health/suppression/adaptive state, and supports reorder/rename and conditional suppression — the dirty-flag DAG pattern (M02) applied to modeling operations.

## In scope

- `PartFeatures` ordered list; reorder/rename.
- Dependency-ordered recompute of the affected tail.
- `HealthStatusEnum`; suppression + `SetSuppressionCondition`.
- `Adaptive`; participants; EOP interaction (M07).

## Out of scope

_None._

## Key API contracts delivered

- `PartFeatures`,`PartFeature`,`Features`,`Feature`
- `HealthStatusEnum`,`ComparisonTypeEnum`,`FeatureDimensions`

## Depends on

M07.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-087](PBI-087-feature-engine.md) | Feature-history recompute engine |
| [PBI-088](PBI-088-suppression-health.md) | Suppression, conditional suppression & health |
| [PBI-089](PBI-089-reorder-rename.md) | Feature reorder, rename & EOP moves |
