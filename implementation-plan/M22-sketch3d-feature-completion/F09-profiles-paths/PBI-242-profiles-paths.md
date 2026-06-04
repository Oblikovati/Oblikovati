---
milestone: M22
feature: F09
pbi: PBI-242
title: Profile3D/Profiles3D + path detection over /api
status: planned
estimate: M
---

# PBI-242 — 3D profiles & paths

**Milestone:** M22  ·  **Feature:** F09 Profiles & Paths

## Goal
Detect chained 3D curves as paths (and planar-closed chains as profiles) and expose them
through `/api` for sweep/loft consumers.

## Scope / work
- `model/sketch/path_3d.go`: chain detection over 3D entities (shared endpoint within
  tol); classify open path vs planar-closed profile; `Profile3D`/`Profiles3D`.
- `/api`: `MethodSketch3DProfiles`/`MethodSketch3DPaths`; `Profile3DInfo`/`Path3DInfo`;
  `contract.Profile3D`; `client` helpers.
- router cases.

## Acceptance criteria
- Unit ≥98%: a 3-segment open polyline → one path; a planar triangle → one closed
  profile; a branch → no single chain.
- Dogfood; round-trip stable ordering; `make ci` green.

## Depends on
PBI-234.
