---
milestone: M06
feature: F06
pbi: PBI-077
title: Profile & region detection
status: planned
estimate: L
---

# PBI-077 — Profile & region detection

**Milestone:** M06 2D/3D Sketching & Constraint Solver  ·  **Feature:** F06 Profiles & Paths

## Goal

Implement region detection from sketch geometry and the `Profile` object (ordered loops, inner/outer classification) that features extrude/revolve.

## Scope / work

- Closed-loop region finding.
- `Profile`/`ProfilePath`/`ProfileEntity`.
- Multi-region & nested-loop handling.

## API contracts (interfaces / enums / collections)

- `Profile`,`Profiles`,`ProfilePath`,`ProfileEntity`

## Acceptance criteria

- A sketch with nested loops yields a profile with correct inner/outer loops.
- An open profile is rejected for solids, allowed for surfaces.

## Depends on

_See feature dependencies._
