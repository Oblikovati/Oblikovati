---
milestone: M11
feature: F01
pbi: PBI-117
title: AssemblyComponentDefinition container
status: planned
estimate: M
---

# PBI-117 — AssemblyComponentDefinition container

**Milestone:** M11 Assembly Modeling & Instancing  ·  **Feature:** F01 Assembly Component Definition

## Goal

Implement the assembly content object exposing the occurrences collection, structure, bounding boxes, and hooks for constraints/representations.

## Scope / work

- `Occurrences` collection.
- Range boxes; geometry version.
- Wire to `AssemblyDocument` (M03).

## API contracts (interfaces / enums / collections)

- `AssemblyComponentDefinition`,`ComponentOccurrences`

## Acceptance criteria

- An assembly document exposes its occurrences and bounding box.

## Depends on

_See feature dependencies._
