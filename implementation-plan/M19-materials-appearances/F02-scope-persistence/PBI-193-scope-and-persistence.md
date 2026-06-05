# PBI-193 — Asset scope tiers + document embedding + persistence

> **Backfilled from shipped code 2026-06-04** (REPORT.md D-03). Grade: **M✅ · G n/a · U n/a**.

## Goal

Built-in / project-library / document-embedded asset tiers that persist and share.

## Scope / work

- `AssetSet`/`Library` containers; duplicate-and-edit custom assets.
- `.obk` round-trip of embedded assets (`recipe.go`); project DesignData library
  shared across a project's documents (`store.go`).

## Acceptance criteria

- Custom appearance survives `.obk` save→reopen (`recipe_test.go`).
- A second document resolves a project-library asset (`library_test.go`,
  `store_test.go`).

## Depends on

PBI-192, ADR-0020 (.obk format).
