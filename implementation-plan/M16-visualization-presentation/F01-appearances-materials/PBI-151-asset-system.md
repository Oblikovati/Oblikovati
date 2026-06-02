---
milestone: M16
feature: F01
pbi: PBI-151
title: Asset system, appearances & materials
status: planned
estimate: L
---

# PBI-151 — Asset system, appearances & materials

**Milestone:** M16 Visualization, Appearances, Styles & Presentations  ·  **Feature:** F01 Appearances & Materials

## Goal

Implement the asset model (typed values/textures, libraries, categories) backing appearances and materials, including physical material properties.

## Scope / work

- `Asset` typed values & textures; libraries/categories.
- `Appearance` (visual) and `Material` (physical+visual).
- Physical props (density/modulus) for mass (M18).

## API contracts (interfaces / enums / collections)

- `Asset(s)`,`AssetLibrary(ies)`,`AssetValue`,`AssetTexture`,`Material(s)`,`Appearance`

## Acceptance criteria

- A material assigns visual appearance and feeds physical mass properties.

## Depends on

_See feature dependencies._
