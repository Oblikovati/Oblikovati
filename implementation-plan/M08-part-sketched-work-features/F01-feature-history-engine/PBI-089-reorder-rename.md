---
milestone: M08
feature: F01
pbi: PBI-089
title: Feature reorder, rename & EOP moves
status: planned
estimate: M
---

# PBI-089 — Feature reorder, rename & EOP moves

**Milestone:** M08 Part Modeling: Sketched & Work Features  ·  **Feature:** F01 Feature History Engine

## Goal

Implement feature reordering, renaming, and moving features relative to the end-of-part marker, with correct re-evaluation.

## Scope / work

- Reorder with dependency validation.
- Rename (id-stable).
- `SetEndOfPart(before)` per feature.

## API contracts (interfaces / enums / collections)

- `PartFeature.Name`,`SetEndOfPart`, reorder API

## Acceptance criteria

- Reordering past a dependency is rejected; valid reorders re-evaluate correctly.

## Depends on

_See feature dependencies._
