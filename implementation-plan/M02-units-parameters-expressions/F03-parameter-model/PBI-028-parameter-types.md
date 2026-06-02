---
milestone: M02
feature: F03
pbi: PBI-028
title: Parameter types & collections (model/user/reference/derived/table)
status: planned
estimate: M
---

# PBI-028 — Parameter types & collections (model/user/reference/derived/table)

**Milestone:** M02 Units, Parameters & Expressions  ·  **Feature:** F03 Parameter Model

## Goal

Implement the parameter subtypes and their collections with type-appropriate set restrictions (e.g. reference parameters are read-only-ish).

## Scope / work

- Each `ParameterTypeEnum` variant's semantics.
- `Parameters`/`ModelParameters`/`UserParameters` with `Add`.
- Naming uniqueness.

## API contracts (interfaces / enums / collections)

- `Parameters`,`ModelParameters`,`UserParameters`,`ReferenceParameter`,`DerivedParameter`,`TableParameter`

## Acceptance criteria

- User parameters can be added/edited; reference/derived enforce read-only as specified.
- Lookup by name works.

## Depends on

_See feature dependencies._
