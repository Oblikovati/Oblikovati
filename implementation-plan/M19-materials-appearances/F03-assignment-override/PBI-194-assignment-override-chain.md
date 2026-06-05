# PBI-194 — Assignment store + override chain by reference key

> **Backfilled from shipped code 2026-06-04** (REPORT.md D-03). Grade: **M✅ · G n/a · U via F07**.

## Goal

Assign material/appearance to a body, surviving recompute, with a defined override chain.

## Scope / work

- Assignment store keyed by `identity.RefKey` (survives recompute/reopen).
- Override chain: explicit body appearance > material's appearance > default.

## Acceptance criteria

- Assignment re-binds after recompute (`assignment_test.go`).
- Override precedence resolves to the explicit body appearance when set.

## Depends on

PBI-192, M03 reference keys.
