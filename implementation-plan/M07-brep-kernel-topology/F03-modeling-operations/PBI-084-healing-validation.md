---
milestone: M07
feature: F03
pbi: PBI-084
title: Geometry healing & validation
status: planned
estimate: L
---

# PBI-084 — Geometry healing & validation

**Milestone:** M07 B-Rep Modeling Kernel & Topology  ·  **Feature:** F03 Boolean & Modeling Operations

## Goal

Implement validation (manifold/orientation/gap checks) and healing (sew gaps, fix orientation) invoked after risky operations and on import.

## Scope / work

- Validity checks.
- Sew/stitch & gap closing.
- Self-intersection detection.

## API contracts (interfaces / enums / collections)

- kernel validate/heal ops

## Acceptance criteria

- An imported open shell can be stitched into a valid body or reported precisely.

## Depends on

_See feature dependencies._
