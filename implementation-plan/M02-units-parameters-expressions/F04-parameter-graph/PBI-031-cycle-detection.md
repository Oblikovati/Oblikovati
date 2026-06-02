---
milestone: M02
feature: F04
pbi: PBI-031
title: Cycle detection & health on the parameter DAG
status: planned
estimate: S
---

# PBI-031 — Cycle detection & health on the parameter DAG

**Milestone:** M02 Units, Parameters & Expressions  ·  **Feature:** F04 Parameter Dependency Graph

## Goal

Reject cyclic dependencies at edit time and mark affected parameters sick instead of crashing.

## Scope / work

- Cycle detection on edge insert.
- Health-status marking; undefined-reference handling.

## API contracts (interfaces / enums / collections)

- `HealthStatusEnum`

## Acceptance criteria

- A cyclic expression is rejected with the parameter marked sick.
- Undefined reference → sick, not exception.

## Depends on

_See feature dependencies._
