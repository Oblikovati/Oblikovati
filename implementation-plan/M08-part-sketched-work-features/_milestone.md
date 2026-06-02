---
milestone: M08
name: Part Modeling: Sketched & Work Features
status: planned
---

# M08 — Part Modeling: Sketched & Work Features

The heart of part modeling: the feature-history recompute engine and the primary sketched features (extrude, revolve, sweep, loft, coil, rib) plus work features (datum planes/axes/points) and derived/reference features. Every feature follows the Definition→Add→Feature triangle and participates in dependency-ordered recompute with health and suppression.

## Goals

- A feature-history engine with ordering, recompute, health, and suppression.
- Work features defined by relationships and usable as feature inputs.
- The core sketched features, each with its Definition object and extents.
- Derived/reference/imported features bringing external geometry in.

## In scope

- `PartFeatures` engine; ordering/reorder/rename; recompute; `HealthStatusEnum`; suppression + conditions.
- `WorkPlane`/`WorkAxis`/`WorkPoint`/`UCS`.
- `ExtrudeFeature`/`RevolveFeature`/`SweepFeature`/`LoftFeature`/`CoilFeature`/`RibFeature` + Definitions/extents.
- Derived part/component; `ReferenceFeatures`; `ImportedComponent`.

## Out of scope (handled elsewhere)

- Dress-up & pattern features (M09).
- Surfacing features (M10).

## Exit criteria

- Extrude/revolve/sweep/loft/coil create valid solids and recompute on parameter change.
- A feature can be suppressed (incl. conditionally), reordered, renamed, and goes sick on lost input.
- Work features defined by relationships drive dependent features.

## Depends on

M06, M07

## Features

| ID | Feature | PBIs | Summary |
|----|---------|:----:|---------|
| **F01** | [Feature History Engine](F01-feature-history-engine/_feature.md) | 3 | Ordered features, recompute, health, suppression, reorder. |
| **F02** | [Work Features (Datums)](F02-work-features/_feature.md) | 2 | Datum planes, axes, points, and UCS by relationship. |
| **F03** | [Sketched Features](F03-sketched-features/_feature.md) | 5 | Extrude, revolve, sweep, loft, coil, rib. |
| **F04** | [Derived & Reference Features](F04-derived-reference-features/_feature.md) | 2 | Derived parts/components and imported geometry as features. |
