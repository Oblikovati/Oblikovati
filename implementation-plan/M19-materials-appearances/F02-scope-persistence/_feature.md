# M19 · F02 — Scope tiers: project library + document embedding + persistence

> **Backfilled 2026-06-04 from shipped code.** See REPORT.md D-03.

## Scope (in)

Asset `Source` tiers (built-in / project-library / document-embedded); duplicate-and-edit
custom assets; `.obk` round-trip of embedded assets; project DesignData library sharing
across documents.

## Code (as built)

`model/material/{assetset.go,library.go,store.go,recipe.go}` (+ `*_test.go`).

## PBIs

| PBI | Title | Grade |
|-----|-------|-------|
| [PBI-193](PBI-193-scope-and-persistence.md) | Asset scope tiers + embedding + persistence | M✅ G n/a U n/a |
