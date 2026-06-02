---
milestone: M07
feature: F01
name: Topology Model
status: planned
---

# M07 · F01 — Topology Model

The boundary-representation topology: solid/surface bodies and their faces, edges, vertices, loops, and edge-uses, with adjacency queries and reference-key identity.

## In scope

- `SurfaceBody`/`SurfaceBodies`; solid vs surface.
- `Face`/`Faces`,`Edge`/`Edges`,`Vertex`/`Vertices`,`Loop`/`EdgeUse`.
- Adjacency queries; geometry access; reference keys.

## Out of scope

_None._

## Key API contracts delivered

- `SurfaceBody`,`SurfaceBodies`,`Face`,`Faces`,`Edge`,`Edges`,`Vertex`,`Vertices`,`EdgeLoop`,`EdgeUse`
- `IRxSurfaceBody`,`IRxAlternateSurfaceBody`

## Depends on

M01,M03.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-079](PBI-079-brep-topology.md) | B-rep topology: bodies/faces/edges/vertices/loops |
| [PBI-080](PBI-080-topology-geometry.md) | Topology↔geometry binding & containers |
