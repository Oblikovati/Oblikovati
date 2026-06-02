---
milestone: M02
feature: F03
pbi: PBI-027
title: Parameter base: expression/value/model-value/units
status: planned
estimate: M
---

# PBI-027 — Parameter base: expression/value/model-value/units

**Milestone:** M02 Units, Parameters & Expressions  ·  **Feature:** F03 Parameter Model

## Goal

Implement the parameter value triad (Expression → Value in db units → ModelValue after tolerance) with units, comment, visibility, and key flags.

## Scope / work

- `Expression`/`Value`/`ModelValue`/`Units`.
- `Comment`,`IsKey`,`Visible`,`InUse`.
- `Delete`, health status.

## API contracts (interfaces / enums / collections)

- `Parameter`,`ParameterTypeEnum`,`HealthStatusEnum`

## Acceptance criteria

- Setting `Expression` updates `Value`/`ModelValue`.
- Setting `Value` is equivalent to a constant expression.

## Depends on

_See feature dependencies._
