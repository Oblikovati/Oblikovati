---
milestone: M09
feature: F02
name: Hole & Boss Features
status: planned
---

# M09 · F02 — Hole & Boss Features

The hole feature family with placement (linear/concentric/on-point/sketch), hole types (simple/counterbore/countersink/spotface), and tapped-thread data that feeds hole tables and drawings, plus boss features for molded parts.

## In scope

- `HoleFeature` types & placements.
- Tap/thread data; depth/termination.
- `BossFeature` (thread/head bosses).

## Out of scope

_None._

## Key API contracts delivered

- `HoleFeature`,`HoleFeatures`,`HoleDefinition`,`HoleTapInfo`
- `BossFeature(s)`,`HolePlacementDefinition`

## Depends on

M08.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-102](PBI-102-hole-feature.md) | Hole feature (types, placement, tap) |
| [PBI-103](PBI-103-boss-feature.md) | Boss features |
