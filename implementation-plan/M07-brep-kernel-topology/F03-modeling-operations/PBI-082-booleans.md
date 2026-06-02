---
milestone: M07
feature: F03
pbi: PBI-082
title: Boolean operations (join/cut/intersect/new-body)
status: planned
estimate: XL
---

# PBI-082 — Boolean operations (join/cut/intersect/new-body)

**Milestone:** M07 B-Rep Modeling Kernel & Topology  ·  **Feature:** F03 Boolean & Modeling Operations

## Goal

Implement robust boolean operations between solid bodies (the `PartFeatureOperationEnum` semantics) with correct topology and reference-key continuity.

## Scope / work

- Join/cut/intersect/new-body.
- Coincident-face & sliver handling.
- Reference-key derivation through booleans.

## API contracts (interfaces / enums / collections)

- `PartFeatureOperationEnum`, kernel boolean ops

## Acceptance criteria

- A∪B, A−B, A∩B produce valid manifold solids.
- Result-face reference keys remain rebindable after edits.

## Depends on

_See feature dependencies._
