---
milestone: M02
feature: F01
pbi: PBI-023
title: UnitsTypeEnum & dimensioned quantity model
status: planned
estimate: M
---

# PBI-023 — UnitsTypeEnum & dimensioned quantity model

**Milestone:** M02 Units, Parameters & Expressions  ·  **Feature:** F01 Unit System

## Goal

Define the unit taxonomy and the rule that every numeric value is a dimensioned quantity stored in database units.

## Scope / work

- Unit categories & members.
- Quantity = (value in db units, unit type).
- Boolean/text as degenerate unit types.

## API contracts (interfaces / enums / collections)

- `UnitsTypeEnum`

## Acceptance criteria

- Every quantity reports a unit type.
- Geometry stored in cm/radians regardless of display.

## Depends on

_See feature dependencies._
