---
milestone: M00
feature: F02
pbi: PBI-007
title: Variant handle down-branching helpers
status: planned
estimate: S
---

# PBI-007 — Variant handle down-branching helpers

**Milestone:** M00 Platform Foundation & Interop  ·  **Feature:** F02 Core Object Model & RTTI

## Goal

Provide ergonomic, safe pattern for callers to test and cast `object` handles by `Type` (selection results, generic containers).

## Scope / work

- `Type`-based type tests and safe casts.
- Helpers for common 'is this an Edge/Face/Feature' checks.

## API contracts (interfaces / enums / collections)

- Casting helpers over `ObjectTypeEnum`

## Acceptance criteria

- A picked `object` can be classified and cast without exceptions on mismatch.

## Depends on

_See feature dependencies._
