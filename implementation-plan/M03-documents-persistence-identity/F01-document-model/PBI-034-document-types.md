---
milestone: M03
feature: F01
pbi: PBI-034
title: Document specializations & content exposure
status: planned
estimate: M
---

# PBI-034 — Document specializations & content exposure

**Milestone:** M03 Documents, Persistence & Identity  ·  **Feature:** F01 Document Model & Types

## Goal

Implement the four document specializations, each exposing its content object and type-specific surface.

## Scope / work

- Part→`PartComponentDefinition`(M07); Assembly→`AssemblyComponentDefinition`(M11).
- Drawing→sheets(M14); Presentation→explosions(M16).
- Branch on `DocumentTypeEnum`.

## API contracts (interfaces / enums / collections)

- `PartDocument`,`AssemblyDocument`,`DrawingDocument`,`PresentationDocument`

## Acceptance criteria

- Each type exposes its content object (stub until later milestones).
- Type discrimination is correct.

## Depends on

_See feature dependencies._
