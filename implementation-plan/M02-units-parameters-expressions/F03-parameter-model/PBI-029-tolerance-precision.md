---
milestone: M02
feature: F03
pbi: PBI-029
title: Tolerance, precision & display format
status: planned
estimate: S
---

# PBI-029 — Tolerance, precision & display format

**Milestone:** M02 Units, Parameters & Expressions  ·  **Feature:** F03 Parameter Model

## Goal

Implement engineering tolerance (the model value computed from expression + tolerance), display precision, and parameter display format.

## Scope / work

- `Tolerance` object & `ModelValueType`.
- `Precision`,`DisplayFormat`,`CustomPropertyFormat`.

## API contracts (interfaces / enums / collections)

- `Tolerance`,`ModelValueTypeEnum`,`ParameterDisplayFormatEnum`,`CustomPropertyFormat`

## Acceptance criteria

- `ModelValue` reflects tolerance per `ModelValueType`.
- Precision affects display only.

## Depends on

_See feature dependencies._
