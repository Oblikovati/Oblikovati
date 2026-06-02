---
milestone: M07
feature: F01
pbi: PBI-079
title: B-rep topology: bodies/faces/edges/vertices/loops
status: planned
estimate: XL
---

# PBI-079 — B-rep topology: bodies/faces/edges/vertices/loops

**Milestone:** M07 B-Rep Modeling Kernel & Topology  ·  **Feature:** F01 Topology Model

## Goal

Implement the topological data model with full adjacency (face↔edge↔vertex↔loop), solid/surface distinction, and reference-key identity on every entity.

## Scope / work

- Topology structure & ownership.
- Adjacency queries (faces of edge, edges of face, etc.).
- `GetReferenceKey` on all entities.

## API contracts (interfaces / enums / collections)

- `SurfaceBody`,`Face`,`Edge`,`Vertex`,`EdgeLoop`,`EdgeUse`,`Faces`,`Edges`

## Acceptance criteria

- Adjacency queries are consistent and complete.
- Each entity yields a reference key that rebinds post-recompute.

## Depends on

_See feature dependencies._
