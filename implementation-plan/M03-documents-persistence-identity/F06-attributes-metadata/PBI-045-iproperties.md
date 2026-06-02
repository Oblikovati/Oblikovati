---
milestone: M03
feature: F06
pbi: PBI-045
title: iProperties / PropertySets
status: planned
estimate: M
---

# PBI-045 — iProperties / PropertySets

**Milestone:** M03 Documents, Persistence & Identity  ·  **Feature:** F06 Attributes & Metadata

## Goal

Implement document-level structured properties (summary/design-tracking/custom) and the parameter→property bridge (`ExposedAsProperty`).

## Scope / work

- Standard + custom property sets.
- `ExposedAsProperty` promotion of parameters.
- Feed to BOM/drawing later.

## API contracts (interfaces / enums / collections)

- `PropertySets`,`PropertySet`,`Property`

## Acceptance criteria

- Custom properties persist and are queryable.
- An exposed parameter appears as a custom property.

## Depends on

_See feature dependencies._
