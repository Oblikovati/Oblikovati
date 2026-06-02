---
milestone: M07
name: B-Rep Modeling Kernel & Topology
status: planned
---

# M07 — B-Rep Modeling Kernel & Topology

The geometric modeling kernel and its topological model: bodies, faces, edges, vertices, loops, and their evaluators, plus the boolean/modeling operations features are built on, and the `PartComponentDefinition` container that owns the evaluated result. This is where transient geometry (M01) acquires identity (M03) and becomes persistent model topology.

## Goals

- A complete B-rep topology model with queries and evaluators.
- Boolean and core modeling operations (union/subtract/intersect, tessellation, healing).
- The part component-definition container owning bodies, bounding boxes, and rollback state.

## In scope

- `SurfaceBody`/`Face`/`Edge`/`Vertex`/`Loop`/`EdgeUse` + collections.
- Face/edge/curve/surface evaluators.
- Boolean ops; tessellation; healing/validation.
- `PartComponentDefinition`; bodies; range boxes; `EndOfPart`.

## Out of scope (handled elsewhere)

- Specific feature recipes (M08/M09).
- Surfacing features (M10).

## Exit criteria

- A body can be constructed, queried (faces/edges/vertices), evaluated, and boolean-combined.
- Topology entities carry reference keys (M03) and survive a recompute.
- The part definition reports bodies and bounding boxes and supports rollback.

## Depends on

M01, M03

## Features

| ID | Feature | PBIs | Summary |
|----|---------|:----:|---------|
| **F01** | [Topology Model](F01-topology-model/_feature.md) | 2 | Bodies, faces, edges, vertices, loops and queries. |
| **F02** | [Geometry Evaluators](F02-geometry-evaluators/_feature.md) | 1 | Face/edge/curve/surface parametric evaluators. |
| **F03** | [Boolean & Modeling Operations](F03-modeling-operations/_feature.md) | 3 | Union/subtract/intersect, tessellation, healing. |
| **F04** | [Part Component Definition Container](F04-part-component-definition/_feature.md) | 2 | The part content object: bodies, boxes, rollback, version. |
