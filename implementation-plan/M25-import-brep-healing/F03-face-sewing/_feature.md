---
milestone: M25
feature: F03
name: Face sewing into watertight shells
status: planned
---

# M25 · F03 — Face sewing into watertight shells

Build the shared-edge topology the raw STEP import lacks: identify which faces share an edge (by 3D
proximity within tolerance) and stitch them into shells, so each interior edge is referenced by
exactly its two faces and meshing/operations see a coherent watertight body — not a loose bag of faces.

## In scope

- Match boundary edges across faces by 3D proximity (within the F02 edge tolerance) and merge them
  into single shared edges with two coedges (one per face).
- Assemble connected faces into shells; report unshared (genuinely open / free) edges.

## Out of scope

- Orientation + validation (F04).
- Self-intersection repair (out of milestone scope).

## Key API contracts delivered

- (internal) edge-matching + shell assembly in the heal path; shared-edge adjacency.

## Depends on

F02 (snapped edges + tolerance), `kernel/topo` (shell/edge/coedge).

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-326](PBI-326-edge-matching.md) | Match + merge shared edges across faces by proximity |
| [PBI-327](PBI-327-shell-assembly.md) | Assemble faces into shells; report free edges |
