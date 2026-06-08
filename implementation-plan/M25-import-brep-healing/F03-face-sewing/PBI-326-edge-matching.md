---
milestone: M25
feature: F03
pbi: PBI-326
title: Match + merge shared edges across faces by proximity
status: planned
estimate: L
---

# PBI-326 — Match + merge shared edges across faces by proximity

**Milestone:** M25 Imported B-Rep Healing  ·  **Feature:** F03 Face sewing into watertight shells

## Goal

Find pairs of face-boundary edges that are the same physical edge (coincident within tolerance) and
merge them into one shared edge with a coedge per face.

## Scope / work

- Index boundary edges spatially (by endpoint positions + midpoint); pair edges whose endpoints +
  curve coincide within the edge tolerance (handle reversed orientation).
- Merge each matched pair into a single `Edge` referenced by two `CoEdge`s (the two faces), using
  the F02 shared polyline; keep each face's own pcurve.
- Leave unmatched edges as free/open boundary.

## API contracts (interfaces / enums / collections)

- (internal) edge-match index + merge; shared `Edge` with two coedges.

## Acceptance criteria

- EDF.STEP: the expected number of edges are matched + merged (each interior edge shared by exactly
  two faces); a synthetic two-face patch sews into one shared edge.
- Reversed-orientation edge pairs match correctly.
- No false merges across a genuine open boundary; OCC oracle green; lint clean.

## Depends on

PBI-324/325 (snapped, clean edges), `kernel/topo`.
