# PBI-192 — Appearance/Material asset object model + built-in catalog

> **Backfilled from shipped code 2026-06-04** (REPORT.md D-03). Grade: **M✅ · G n/a · U n/a**.

## Goal

Typed PBR `Appearance` and physical `Material` assets with a built-in catalog.

## Scope / work

- `Appearance`: albedo, metallic, roughness, emissive, opacity, source, id, name.
- `Material`: density, `Mechanical`/`Thermal`/`Electrical`, `IsotropyClass` +
  `AnisotropicElastic`, `AppearanceID`, source, id, name.
- Built-in catalog seeded at init (`builtin.go`, embedded `catalog/`).

## API contracts

`api/contract.Appearance`, `api/contract.Material`; `api/types.{AssetSource,Mechanical,
Thermal,Electrical,IsotropyClass,AnisotropicElastic,Rgba}`.

## Acceptance criteria

- `model/material` types satisfy the contracts (compile-time assertions).
- Built-in catalog loads; `catalog_test.go` green; colors parse (`builtin.go`).

## Depends on

ADR-0022, ADR-0024, ADR-0025.
