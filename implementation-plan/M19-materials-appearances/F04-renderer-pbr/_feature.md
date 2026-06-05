# M19 · F04 — Renderer PBR surface resolution

> **Backfilled 2026-06-04 from shipped code.** See REPORT.md D-03.

## Scope (in)

Resolve a body's effective appearance → PBR surface values the draw list carries, so the
viewport recolors live on assignment. Solid (non-textured) metallic-roughness only.

## Scope (out)

GGX/IBL shading + image textures (M23 / later); per-face override *rendering* (mesh split).

## Code (as built)

`renderer/drawlist.go` (surface fields), `app/material_ops.go` (resolution into the
session draw list).

## PBIs

| PBI | Title | Grade |
|-----|-------|-------|
| [PBI-195](PBI-195-renderer-pbr-resolution.md) | Appearance → draw-list PBR surface | M✅ G✅ U(viewport, see F07) |
