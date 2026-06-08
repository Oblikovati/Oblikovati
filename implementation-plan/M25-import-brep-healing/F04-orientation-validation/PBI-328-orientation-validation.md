---
milestone: M25
feature: F04
pbi: PBI-328
title: Coherent shell orientation + manifold/closed validation
status: planned
estimate: M
---

# PBI-328 — Coherent shell orientation + manifold/closed validation

**Milestone:** M25 Imported B-Rep Healing  ·  **Feature:** F04 Orientation, validation & oracle gate

## Goal

Make every healed shell consistently outward-oriented and validate it, so downstream consumers
(mesher, mass-props, boolean) get a coherent body.

## Scope / work

- Propagate a consistent orientation over the shell via shared-edge use directions (each interior
  edge traversed oppositely by its two coedges); flip faces to make it coherent.
- Flip the whole shell if its signed volume is negative (outward = positive).
- Validate: 2-manifold (each edge ≤ 2 faces), closed where expected; emit a healing report.

## API contracts (interfaces / enums / collections)

- (internal) shell orientation + manifold/closed validation + report.

## Acceptance criteria

- EDF.STEP shells are coherently outward (signed volume positive, matching OCC sign) and 2-manifold.
- A deliberately mis-oriented synthetic shell is corrected.
- OCC oracle green; lint clean.

## Depends on

PBI-327 (shells).
