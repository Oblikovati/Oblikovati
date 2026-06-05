# M19 · F03 — Assignment & override chain (by reference key)

> **Backfilled 2026-06-04 from shipped code.** See REPORT.md D-03.

## Scope (in)

Assign material/appearance to a body (and the appearance-source override chain:
material's appearance vs explicit body appearance) keyed by **reference key** so the
assignment survives recompute.

## Code (as built)

`model/material/assignment.go` (+ `assignment_test.go`).

## PBIs

| PBI | Title | Grade |
|-----|-------|-------|
| [PBI-194](PBI-194-assignment-override-chain.md) | Assignment store + override chain by ref key | M✅ G n/a U(see F07) |
