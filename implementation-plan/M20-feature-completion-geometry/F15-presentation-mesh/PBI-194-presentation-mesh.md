---
milestone: M20
feature: F15
pbi: PBI-194
title: Presentation mesh & mesh→B-rep
status: planned
estimate: M
---

# PBI-194 — Presentation mesh & mesh→B-rep

**Milestone:** M20 Feature Completion & Geometry Parity  ·  **Feature:** F15 Presentation Mesh Feature

## Goal

A `PresentationMeshFeature` (lightweight display mesh) and conversion of an imported mesh to a B-rep solid.

## Scope / work

`PresentationMeshFeature` wraps a `MeshGeometry` for display-only use (passes the solid through); `ops.MeshToBRep` converts a closed welded mesh to a faceted B-rep solid (one planar face per facet, shared edges/verts).

## API contracts (interfaces / enums / collections)

- `PresentationMeshFeature(s)`; `ops.MeshToBRep`.

## Acceptance criteria

- A tetra mesh converts to a validated 4-face solid
- the presentation-mesh feature carries the mesh without altering the running body
- round-trip.

## Depends on

M10

## Notes

Follows the canonical feature pattern set by Extrude (M08·PBI-092): full
`Definition → Add → Feature(+Proxy)` triangle, `/api` `wire`+`client`+`contract`,
`addin/router` handler, `.obk` round-trip, and executable tests.
