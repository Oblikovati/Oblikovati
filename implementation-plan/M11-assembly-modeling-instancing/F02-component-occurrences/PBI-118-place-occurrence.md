---
milestone: M11
feature: F02
pbi: PBI-118
title: Place/copy occurrences & shared definitions
status: planned
estimate: L
---

# PBI-118 — Place/copy occurrences & shared definitions

**Milestone:** M11 Assembly Modeling & Instancing  ·  **Feature:** F02 Component Occurrences

## Goal

Implement placing a component (creating an occurrence referencing a shared definition) and copying, such that N occurrences share one definition and memory scales with unique parts.

## Scope / work

- `Occurrences.Add(file, position)`/`AddByComponentDefinition`.
- Shared-definition semantics; editing definition updates all instances.
- Delete/replace.

## API contracts (interfaces / enums / collections)

- `ComponentOccurrences.Add*`,`ComponentOccurrence.Definition`

## Acceptance criteria

- Two placements of one part share a definition; editing the part updates both.

## Depends on

_See feature dependencies._
