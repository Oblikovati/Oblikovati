---
milestone: M02
feature: F04
name: Parameter Dependency Graph
status: planned
---

# M02 · F04 — Parameter Dependency Graph

Parameters and expressions form a directed acyclic graph whose edges (`DrivenBy`/`Dependents`) drive recompute. This feature builds the graph, detects cycles at edit time, and propagates dirtiness in dependency order — the same discipline reused by the feature engine.

## In scope

- `DrivenBy`/`Dependents` edges.
- Cycle detection & rejection via health status.
- Dirty-propagation ordering; rename-safe references.

## Out of scope

_None._

## Key API contracts delivered

- `Parameter.Dependents`,`Parameter.DrivenBy`
- `ExpressionList`

## Depends on

F02,F03.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-030](PBI-030-dependency-edges.md) | Dependency edges & DrivenBy/Dependents queries |
| [PBI-031](PBI-031-cycle-detection.md) | Cycle detection & health on the parameter DAG |
| [PBI-032](PBI-032-dirty-propagation.md) | Dirty-propagation & dependency-ordered recompute |
