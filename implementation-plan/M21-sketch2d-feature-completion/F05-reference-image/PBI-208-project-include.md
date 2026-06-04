---
milestone: M21
feature: F05
pbi: PBI-208
title: Project geometry & include
status: planned
estimate: L
---

# PBI-208 — Project geometry & include

**Milestone:** M21  ·  **Feature:** F05 Reference & Image Entities

## Goal

Project model edges, vertices, and work geometry onto the sketch plane as reference
entities that re-bind through recompute (reference keys), plus "include" of edges as
real sketch geometry.

## Scope / work

- **/source:** extend `model/sketch/projection.go` — project an edge/vertex/work-axis/
  work-point onto the sketch plane; store the source `ReferenceKey` so recompute re-derives
  the projected curve; cut-edge projection (section). Health → sick on lost reference.
- **/api:** `MethodSketchProject`, `wire.ProjectGeometryArgs` (sketch + source refs +
  mode include|reference); `client.Sketch.Project/Include`.
- **UI:** project-geometry tool (pick edges/faces), ribbon command, e2e.

## Acceptance criteria

- Dogfood + UI e2e: projecting a box edge yields a sketch line at the right location;
  after a recompute that moves the edge, the projection follows; a lost reference →
  Sick health, no crash. Round-trip preserves the reference. `make ci` green.

## Depends on

PBI-202, M03 reference keys, M08 part edges.
