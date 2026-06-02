---
milestone: M07
feature: F04
pbi: PBI-086
title: Rollback / End-of-Part state
status: planned
estimate: M
---

# PBI-086 — Rollback / End-of-Part state

**Milestone:** M07 B-Rep Modeling Kernel & Topology  ·  **Feature:** F04 Part Component Definition Container

## Goal

Implement the rollback marker (end-of-part) that defines how far the feature program evaluates and supports mid-history editing.

## Scope / work

- `RolledBack` state; EOP marker position.
- Evaluate-up-to-marker semantics (consumed by M08).

## API contracts (interfaces / enums / collections)

- `PartComponentDefinition` rollback API,`EndOfFeatures`

## Acceptance criteria

- Moving the EOP marker re-evaluates the model to that point.

## Depends on

_See feature dependencies._
