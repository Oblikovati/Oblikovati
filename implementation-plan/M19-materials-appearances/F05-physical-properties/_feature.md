# M19 · F05 — Mass / physical properties

> **Backfilled 2026-06-04 from shipped code.** See REPORT.md D-03.

## Scope (in)

Mass / volume / surface-area / centroid readout for a part, using the assigned material's
density. Shared with M18 (analysis) which consumes the same property data.

## Code (as built)

`kernel/ops/massprops.go`; consumed via `app/materials.go` readout. (This is also the
M18·F01 mass-properties enabler — cross-referenced there.)

## PBIs

| PBI | Title | Grade |
|-----|-------|-------|
| [PBI-196](PBI-196-physical-properties.md) | Mass properties from assigned material | M✅ G✅ U(readout, see F07) |
