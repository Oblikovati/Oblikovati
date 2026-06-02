---
milestone: M00
feature: F02
pbi: PBI-006
title: Universal Type/Parent/Application members & navigation
status: planned
estimate: S
---

# PBI-006 — Universal Type/Parent/Application members & navigation

**Milestone:** M00 Platform Foundation & Interop  ·  **Feature:** F02 Core Object Model & RTTI

## Goal

Guarantee every model object can report its type and walk up to its document and the app root without caller-threaded context.

## Scope / work

- Base contract mixed into all objects.
- Parent chain resolution to `Document`/`Application`.

## API contracts (interfaces / enums / collections)

- `Type`,`Parent`,`Application` on all objects

## Acceptance criteria

- From any object, the document and application root are reachable.
- `Type` matches `ObjectTypeEnum`.

## Depends on

_See feature dependencies._
