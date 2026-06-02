---
milestone: M10
feature: F04
name: Mesh, Imported Geometry & Mold
status: planned
---

# M10 · F04 — Mesh, Imported Geometry & Mold

Mesh feature handling (imported tessellated geometry as first-class mesh features) and mold tooling (core/cavity split for injection-mold design).

## In scope

- MeshFeature/MeshFeatureSet; mesh topology.
- NonParametricBase from import.
- CoreCavity (mold split).

## Out of scope

_None._

## Key API contracts delivered

- `MeshFeature(s)`,`MeshFeatureSet(s)`,`MeshFeatureEntity`,`MeshFace`,`MeshEdge`,`MeshVertex`
- `CoreCavityFeature(s)`,`NonParametricBaseFeature(s)`

## Depends on

F01.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-115](PBI-115-mesh-features.md) | Mesh features & mesh topology |
| [PBI-116](PBI-116-core-cavity.md) | Mold core/cavity tooling |
