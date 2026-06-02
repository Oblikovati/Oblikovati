---
milestone: M00
feature: F01
pbi: PBI-002
title: Object handle wrappers with preserved identity & equality
status: planned
estimate: M
---

# PBI-002 — Object handle wrappers with preserved identity & equality

**Milestone:** M00 Platform Foundation & Interop  ·  **Feature:** F01 Native/Managed Interop Layer

## Goal

Wrap native objects in managed handles such that the same native object always yields an equal managed wrapper (reference identity preserved across calls).

## Scope / work

- Handle table mapping native pointer ↔ managed wrapper.
- `Equals`/`GetHashCode` based on native identity.
- Lifetime: weak references + native ref-counting.

## API contracts (interfaces / enums / collections)

- (internal) NativeHandle, HandleRegistry

## Acceptance criteria

- Two API calls returning the same native object produce equal wrappers.
- No managed wrapper outlives its native object without a defined error.

## Depends on

_See feature dependencies._
