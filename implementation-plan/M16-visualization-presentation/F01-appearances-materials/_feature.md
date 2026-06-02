---
milestone: M16
feature: F01
name: Appearances & Materials
status: planned
---

# M16 · F01 — Appearances & Materials

The asset system underlying appearances (visual: color/reflectivity/texture) and materials (physical: density/modulus + visual), with libraries, per-entity overrides, and the appearance-source resolution that decides what an object shows.

## In scope

- `Asset`/`AssetLibrary(ies)`/`AssetValue`/`AssetTexture`.
- `Appearance`/`Material`; physical properties.
- `AppearanceSourceType` override resolution.

## Out of scope

_None._

## Key API contracts delivered

- `Asset`,`Assets`,`AssetLibrary(ies)`,`AssetValue`,`AssetTexture`,`AssetCategory(ies)`
- `Material`,`Materials`,`MaterialAsset`,`Appearance`,`AppearanceSourceTypeEnum`,`RenderStyle`

## Depends on

M07.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-151](PBI-151-asset-system.md) | Asset system, appearances & materials |
| [PBI-152](PBI-152-appearance-override.md) | Appearance source resolution & overrides |
