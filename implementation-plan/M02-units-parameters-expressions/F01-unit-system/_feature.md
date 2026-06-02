---
milestone: M02
feature: F01
name: Unit System
status: planned
---

# M02 · F01 — Unit System

A total unit system: every value carries a unit type; the kernel works in canonical database units (cm, radians); conversions happen only at the display/parse boundary.

## In scope

- `UnitsTypeEnum` (length/angle/area/mass/text/boolean…).
- Database-unit canonicalization & conversion.
- Document unit preferences.

## Out of scope

_None._

## Key API contracts delivered

- `UnitsTypeEnum`
- `UnitsOfMeasure`

## Depends on

M00.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-023](PBI-023-units-typeenum.md) | UnitsTypeEnum & dimensioned quantity model |
| [PBI-024](PBI-024-units-conversion.md) | UnitsOfMeasure: conversion & display formatting |
