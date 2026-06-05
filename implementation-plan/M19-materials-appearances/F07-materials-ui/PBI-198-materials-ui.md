# PBI-198 — Materials window (browser, editors, assign, readout)

> **Backfilled from shipped code 2026-06-04** (REPORT.md D-03).
> Grade: **M✅ · U🟦 (e2e unverified — see REPORT.md §8)**.

## Goal

A user can browse/edit/assign materials & appearances in the head and see the result.

## Scope / work

- `head/ui/materials_window.go`, `appearance_editor.go`, `appearance_tab.go`,
  `preferences_window.go`; `app/materials.go` session bridges.
- Assign-to-selection; appearance editor (albedo/metallic/roughness/emissive/opacity);
  physical-properties readout.

## Acceptance criteria (DoD)

- End-to-end: open Materials window → duplicate appearance → edit albedo → assign to
  selection → **viewport recolors live** → read mass (the milestone exit criterion).
- **Gap:** `head` tests are excluded from the `CGO_ENABLED=0` CI run (REPORT.md §8), so
  the e2e assertion is **not currently verified in CI**. Add a head-test gate before
  grading **U✅**.

## Depends on

PBI-195..197, M05 UI shell, ADR-0021.
