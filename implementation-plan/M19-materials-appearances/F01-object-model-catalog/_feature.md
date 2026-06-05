# M19 · F01 — Appearance & Material object model + built-in catalog

> **Backfilled 2026-06-04 from shipped code** (this milestone was implemented before its
> feature/PBI files existed — see REPORT.md D-03). Grades reflect actual code state.

## Scope (in)

Typed `Appearance` (metallic-roughness PBR: albedo/metallic/roughness/emissive/opacity)
and `Material` (density + mechanical/thermal/electrical, isotropy class + anisotropic
elastic constants, → appearance by id) asset objects, plus a built-in catalog seeded at
init.

## Scope (out)

Image-texture maps; per-document scoping (F02); rendering (F04).

## Code (as built)

`model/material/{appearance.go,material.go,catalog.go,catalog/,builtin.go}`.

## PBIs

| PBI | Title | Grade |
|-----|-------|-------|
| [PBI-192](PBI-192-asset-object-model.md) | Appearance/Material assets + built-in catalog | M✅ G n/a U n/a |
