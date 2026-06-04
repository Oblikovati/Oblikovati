---
milestone: M22
feature: F09
name: Profiles & Paths (3D)
status: planned
---

# M22 · F09 — Profiles & Paths (3D)

`Profile3D`/`Profiles3D` and open-path detection over `/api`: a 3D sketch's chained
curves form a path (open, for sweep/loft) or, when planar and closed, a profile. Exposes
`sketch3d.profiles` / `sketch3d.paths` for the feature engine (M08/M10) to consume.

## Depends on
F02, F03, F04 (the curves to chain), F01.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-242](PBI-242-profiles-paths.md) | Profile3D/Profiles3D + path detection over /api |
