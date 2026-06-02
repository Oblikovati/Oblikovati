---
milestone: M00
feature: F02
pbi: PBI-005
title: ObjectTypeEnum: stable numbered RTTI taxonomy
status: planned
estimate: M
---

# PBI-005 — ObjectTypeEnum: stable numbered RTTI taxonomy

**Milestone:** M00 Platform Foundation & Interop  ·  **Feature:** F02 Core Object Model & RTTI

## Goal

Define the master object-type enumeration with stable, explicit, never-renumbered ids and category ranges, exposed as `Type` on every object.

## Scope / work

- Allocate id ranges per category (geometry, features, assembly, drawing…).
- Codegen/sync mechanism keeping the enum exhaustive.
- Persistence-safe value stability policy.

## API contracts (interfaces / enums / collections)

- `ObjectTypeEnum`

## Acceptance criteria

- Every object returns a correct, stable `Type`.
- Ids are documented as immutable; a renumber test fails the build.

## Depends on

_See feature dependencies._
