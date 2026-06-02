---
milestone: M11
feature: F04
pbi: PBI-122
title: Mirror, copy & substitute components
status: planned
estimate: M
---

# PBI-122 — Mirror, copy & substitute components

**Milestone:** M11 Assembly Modeling & Instancing  ·  **Feature:** F04 Component Patterns, Mirror & Substitution

## Goal

Implement mirror-components (with new mirrored parts where needed), copy-components, and occurrence substitution.

## Scope / work

- `MirrorComponentsDefinition` (mirror vs reuse).
- `CopyComponentsDefinition`.
- `IsSubstituteOccurrence` & substitution.

## API contracts (interfaces / enums / collections)

- `MirrorComponentsDefinition`,`CopyComponentsDefinition`,`DerivedAssemblyOccurrence(s)`

## Acceptance criteria

- Mirroring produces correctly handed components; substitution swaps in a simplified rep.

## Depends on

_See feature dependencies._
