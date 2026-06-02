---
milestone: M09
feature: F04
pbi: PBI-106
title: Combine & split (multi-body)
status: planned
estimate: M
---

# PBI-106 — Combine & split (multi-body)

**Milestone:** M09 Part Modeling: Dress-up & Pattern Features  ·  **Feature:** F04 Modify & Direct-Edit Features

## Goal

Implement multi-body combine (join/cut/intersect between bodies) and split (by plane/surface/sketch).

## Scope / work

- `CombineFeature` operation between bodies.
- `SplitFeature` split body/faces/trim.

## API contracts (interfaces / enums / collections)

- `CombineFeature(s)`,`SplitFeature(s)`

## Acceptance criteria

- Two bodies combine; a body splits into multiple solids.

## Depends on

_See feature dependencies._
