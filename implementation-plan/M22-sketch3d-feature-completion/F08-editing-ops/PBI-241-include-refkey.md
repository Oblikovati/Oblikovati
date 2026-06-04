---
milestone: M22
feature: F08
pbi: PBI-241
title: Include part geometry + GetReferenceKey (3D)
status: planned
estimate: M
---

# PBI-241 — Include + reference keys (3D)

**Milestone:** M22  ·  **Feature:** F08 Editing & Reference Ops

## Goal
Project part edges/vertices/work geometry into the active 3D sketch as reference
entities, and surface reference keys for 3D sketch + entities.

## Scope / work
- `model/sketch/include_3d.go`: `Include(ref)` copies a B-rep edge/vertex (or work
  axis/point) as a construction 3D entity bound by reference key; rebinds on recompute.
- `/api`: `MethodSketch3DInclude` (`Sketch3DIncludeArgs`: target ref key);
  `MethodSketch3DGetReferenceKey`; `client` helpers.
- router cases; UI Include/Project tool + ribbon button.

## Acceptance criteria
- Identity tests: included entity survives recompute; fails honestly (→ sick) when the
  source is lost; survives reload.
- Dogfood; ≥1 UI e2e test; `make ci` green.

## Depends on
PBI-240, M07 topology, M03 reference keys.
