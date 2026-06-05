# PBI-196 — Mass / physical properties from assigned material

> **Backfilled from shipped code 2026-06-04** (REPORT.md D-03). Grade: **M✅ · G✅ · U via F07**.

## Goal

Read mass/volume/area/centroid for a part using the assigned material's density.

## Scope / work

- `kernel/ops/massprops.go` computes volume/centroid/area from the B-rep; mass =
  density × volume.
- Also the M18·F01 enabler (cross-referenced; this is where mass properties live).

## Acceptance criteria

- Known solid → analytic volume/centroid within tolerance.
- Mass tracks the assigned material density.

## Depends on

PBI-194, M07 B-rep. **Cross-ref:** M18·F01 (PBI-165 mass-properties).
