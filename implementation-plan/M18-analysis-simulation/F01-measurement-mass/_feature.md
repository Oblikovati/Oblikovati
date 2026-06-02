---
milestone: M18
feature: F01
name: Measurement & Mass Properties
status: planned
---

# M18 · F01 — Measurement & Mass Properties

Interactive and programmatic measurement (distance/angle/area/loop) and mass-properties computation (volume/mass/center-of-mass/moments of inertia) using material physical properties, plus precise bounding ranges.

## In scope

- `MeasureTools` (distance/angle/area/min-distance).
- `MassProperties` (mass/com/inertia/principal axes).
- Precise/oriented range boxes (M07).

## Out of scope

_None._

## Key API contracts delivered

- `MeasureTools`,`MeasureEvents`,`MassProperties`,`Box`,`OrientedBox`

## Depends on

M07,M16.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-164](PBI-164-measure.md) | Measurement tools |
| [PBI-165](PBI-165-mass-properties.md) | Mass & physical properties |
