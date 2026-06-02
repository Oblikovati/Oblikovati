---
milestone: M03
feature: F05
pbi: PBI-043
title: Reference-loss propagation policy
status: planned
estimate: S
---

# PBI-043 — Reference-loss propagation policy

**Milestone:** M03 Documents, Persistence & Identity  ·  **Feature:** F05 Persistent Identity (Reference Keys)

## Goal

Define the system-wide contract for what happens when a bind fails: consumer goes sick, surfaces for re-selection, never crashes.

## Scope / work

- Standard 'reference lost' result.
- Wiring to `HealthStatusEnum` on features/dimensions.
- User re-selection hook.

## API contracts (interfaces / enums / collections)

- `HealthStatusEnum`, bind-failure result

## Acceptance criteria

- A feature whose input reference is lost goes sick and is fixable, not fatal.

## Depends on

_See feature dependencies._
