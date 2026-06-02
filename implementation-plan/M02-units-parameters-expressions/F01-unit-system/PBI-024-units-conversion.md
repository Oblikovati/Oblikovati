---
milestone: M02
feature: F01
pbi: PBI-024
title: UnitsOfMeasure: conversion & display formatting
status: planned
estimate: M
---

# PBI-024 — UnitsOfMeasure: conversion & display formatting

**Milestone:** M02 Units, Parameters & Expressions  ·  **Feature:** F01 Unit System

## Goal

Implement conversion between database and user units plus parse/format at the boundary, honoring document unit preferences.

## Scope / work

- db↔user conversion per unit type.
- Parse user strings to db units; format db→user.
- `GetStringFromValue`/`GetValueFromString` equivalents.

## API contracts (interfaces / enums / collections)

- `UnitsOfMeasure`

## Acceptance criteria

- Round-trip parse/format is lossless within precision.
- Switching document units changes display, not stored values.

## Depends on

_See feature dependencies._
