---
milestone: M02
feature: F04
pbi: PBI-032
title: Dirty-propagation & dependency-ordered recompute
status: planned
estimate: M
---

# PBI-032 — Dirty-propagation & dependency-ordered recompute

**Milestone:** M02 Units, Parameters & Expressions  ·  **Feature:** F04 Parameter Dependency Graph

## Goal

Propagate change from a parameter to exactly its transitive dependents and recompute them in dependency order.

## Scope / work

- Topological recompute of the affected sub-DAG.
- Hook point for downstream feature dirtying (M08).

## API contracts (interfaces / enums / collections)

- (internal) recompute scheduler

## Acceptance criteria

- Changing one parameter recomputes only its dependents.
- Recompute order respects the DAG.

## Depends on

_See feature dependencies._
