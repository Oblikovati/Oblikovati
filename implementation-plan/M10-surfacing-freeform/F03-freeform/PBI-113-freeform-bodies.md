---
milestone: M10
feature: F03
pbi: PBI-113
title: Freeform (sub-D) bodies & primitives
status: planned
estimate: L
---

# PBI-113 — Freeform (sub-D) bodies & primitives

**Milestone:** M10 Surfacing & Freeform Modeling  ·  **Feature:** F03 Freeform Modeling

## Goal

Implement free-form body creation from primitives and the sub-D topology (faces/edges/vertices/cage).

## Scope / work

- Primitive starts (box/sphere/cylinder/torus/quadball/plane).
- Sub-D control cage; tessellation to B-rep.

## API contracts (interfaces / enums / collections)

- `FreeformFeature(s)`,`FreeformBody`,`FreeformFace/Edge/Vertex`,`FreeformBodies`

## Acceptance criteria

- A free-form primitive is created and converts to a B-rep body.

## Depends on

_See feature dependencies._
