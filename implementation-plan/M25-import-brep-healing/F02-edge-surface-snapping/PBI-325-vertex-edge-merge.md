---
milestone: M25
feature: F02
pbi: PBI-325
title: Merge near-coincident vertices + degenerate edges
status: planned
estimate: S
---

# PBI-325 — Merge near-coincident vertices + degenerate edges

**Milestone:** M25 Imported B-Rep Healing  ·  **Feature:** F02 Edge/surface tolerance snapping

## Goal

Collapse near-coincident vertices and drop degenerate/zero-length edges within the model tolerance,
so the topology is clean before sewing.

## Scope / work

- Merge vertices within tolerance (a spatial weld), rewiring incident edges.
- Drop zero-length / degenerate edges (sphere-pole / sliver artifacts) and fix the loops that used them.
- Tolerance-driven; idempotent.

## API contracts (interfaces / enums / collections)

- (internal) vertex weld + degenerate-edge cleanup in the heal path.

## Acceptance criteria

- A model with duplicated near-coincident vertices welds to the expected unique count; loops stay valid.
- Degenerate edges removed without breaking face loops.
- Clean fixtures unchanged; OCC oracle green; lint clean.

## Depends on

PBI-324; `kernel/topo`.
