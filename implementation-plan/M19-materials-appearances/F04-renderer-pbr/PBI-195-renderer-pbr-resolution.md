# PBI-195 — Appearance → draw-list PBR surface resolution

> **Backfilled from shipped code 2026-06-04** (REPORT.md D-03). Grade: **M✅ · G✅ · U via F07**.

## Goal

The viewport shows a body's assigned appearance and recolors live on reassignment.

## Scope / work

- Draw-list items carry resolved PBR surface values (`renderer/drawlist.go`).
- `app/material_ops.go` resolves the effective appearance per body into the frame.
- Solid metallic-roughness only (no GGX/IBL/textures — out of scope, → M23/later).

## Acceptance criteria

- Assigning an appearance changes the draw item's surface values (metamorphic test on
  the null backend).

## Depends on

PBI-194, M05 client graphics, ADR-0022.
