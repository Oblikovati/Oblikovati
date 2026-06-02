---
milestone: M06
feature: F06
name: Profiles & Paths
status: planned
---

# M06 · F06 — Profiles & Paths

The boundary between sketching and modeling: region detection that turns sketch loops into `Profile` objects (with inner/outer loops) and path extraction for sweeps — the clean interface the feature engine consumes.

## In scope

- Region/loop detection; `Profile`/`ProfilePath`.
- Inner vs outer loops; multi-region profiles.
- Paths for sweep/loft rails.

## Out of scope

_None._

## Key API contracts delivered

- `Profile`,`Profiles`,`ProfilePath`,`ProfileEntity`
- `Path`,`Path3D`

## Depends on

F05.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-077](PBI-077-profiles.md) | Profile & region detection |
| [PBI-078](PBI-078-paths.md) | Paths for sweeps & lofts |
