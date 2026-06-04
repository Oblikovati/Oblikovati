# ADR-0024 — Embedded built-in material & appearance catalog (YAML, go:embed)

**Status:** accepted (user decision, 2026-06-04) · **Relates to:**
[ADR-0022](ADR-0022-materials-appearances.md) (materials & appearances subsystem),
[ADR-0020](ADR-0020-yaml-git-friendly-document-format.md) (YAML asset recipes),
[ADR-0023](ADR-0023-viewport-display-modes.md) (Realistic PBR display mode that
renders these appearances).

## Context

ADR-0022 shipped the materials/appearances subsystem with a placeholder built-in
catalog: four materials and six appearances hand-coded as Go literals in
`model/material/builtin.go`. To make Realistic mode useful we want a real,
industry-representative library — periodic-table metals (Z ≤ 83), steel and aluminium
alloys, plastics/resins, woods and composites — each with sourced mechanical, thermal
and electrical properties and a physically-based PBR appearance.

That is ~100+ entries. Encoding them as Go literals would blow past CLAUDE.md's
500-line file limit, bury reviewable data in code, and make value diffs noisy. The
asset data already has a canonical persisted shape — `AppearanceRecipe` /
`MaterialRecipe` YAML (ADR-0020) — and a tested converter (`recipeToAppearance` /
`recipeToMaterial`).

## Decision

1. **The built-in catalog ships as embedded YAML, not Go literals.** Category files live
   in `model/material/catalog/` (`01-metals.yaml`, `02-steels.yaml`, `03-aluminum.yaml`,
   `04-plastics.yaml`, `05-woods.yaml`, `06-composites.yaml`), embedded via `go:embed`
   and loaded by `loadCatalog` in `catalog.go`. The numeric prefix fixes display order.
   Files reuse the existing `RecipeData` shape, so there is **no new type or wire
   surface** — this is implementation-only (no `../Oblikovati.API` change).

2. **Built-ins load as `SourceBuiltin` (read-only).** `seedBuiltins` adds the neutral
   `default` appearance (kept in code so `DefaultAppearanceID` is guaranteed present even
   if a file is edited), then folds every catalog file in. A malformed file or colour is
   a **build defect**: it panics, caught by the package tests — never a runtime branch.

3. **Admission rule: reliable data only.** An entry ships only when every mechanical,
   thermal and electrical field has a sourced value (`TestEveryMaterialHasReliableProperties`
   enforces non-zero). Reactive/liquid metals and grades without dependable structural
   data are omitted rather than guessed. Each file header cites its sources (ASM, CRC,
   MatWeb, USDA FPL Wood Handbook, CMH-17, manufacturer datasheets).

4. **Appearances are physically-based for the Realistic shader.** Metals use measured F0
   reflectance base colours (Lagarde/UE4/Filament chart) authored in **sRGB** — the
   shader linearises albedo (`mesh.frag`: `f0 = toLinear(albedo)`, ADR-0023) — with
   `metallic = 1.0` and a finish-dependent roughness. Dielectrics use `metallic = 0.0`;
   optically clear grades (acrylic, PC, PETG…) carry `opacity < 1.0`. Grain/weave texture
   maps remain deferred (ADR-0022 §2): appearances are flat diffuse tones for now.

## Consequences

- The library grows from 4/6 to ~115 materials / ~88 appearances with no code-size or
  type-surface cost; values are reviewed as data and diff cleanly.
- A few legacy ids (`steel`, `aluminum-6061`, `aluminum`, `oak`, `abs-black`) are kept as
  real entries so existing cross-package tests and assignments stay valid.
- Catalog correctness is now test-guarded (count, unique ids, complete properties, PBR
  ranges, required ids). Editing a YAML file re-runs through the same gates.
- Future work: texture-map appearances (needs binary document sections + UVs), and
  per-anisotropy-direction material data for woods/composites (current values are the
  conventional single-axis engineering figures, documented in each file header).
