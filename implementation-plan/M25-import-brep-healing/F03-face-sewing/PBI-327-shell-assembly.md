---
milestone: M25
feature: F03
pbi: PBI-327
title: Assemble faces into shells; report free edges
status: planned
estimate: M
---

# PBI-327 — Assemble faces into shells; report free edges

**Milestone:** M25 Imported B-Rep Healing  ·  **Feature:** F03 Face sewing into watertight shells

## Goal

Group connected (now edge-sharing) faces into shells and report any remaining free edges, so the
healed body is a coherent shell (closed where it should be).

## Scope / work

- Connected-component the faces over the shared-edge adjacency into shells.
- Classify each shell closed (every edge shared by 2 faces) or open (has free edges); report free
  edges + their count for validation.
- Attach shells to the body in place of the loose face set.

## API contracts (interfaces / enums / collections)

- (internal) shell assembly; free-edge report.

## Acceptance criteria

- EDF.STEP: its solids assemble into closed shells with **0 free edges** (a watertight body), or the
  free edges are reported (genuine openings).
- A synthetic open surface stays open (free edges reported, not falsely closed).
- OCC oracle green; lint clean.

## Depends on

PBI-326 (shared edges).
