---
milestone: M11
feature: F01
name: Assembly Component Definition
status: planned
---

# M11 · F01 — Assembly Component Definition

The `AssemblyComponentDefinition` that owns the assembly's occurrences, structure, and (later) constraints/representations — the assembly analogue of the part definition.

## In scope

- `AssemblyComponentDefinition`.
- Occurrences container; structure.
- Bounding boxes; representations container hook (M12).

## Out of scope

_None._

## Key API contracts delivered

- `AssemblyComponentDefinition`,`AssemblyComponentDefinitions`,`_AssemblyComponentDefinition`
- `ComponentOccurrences`

## Depends on

M07.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-117](PBI-117-assembly-definition.md) | AssemblyComponentDefinition container |
