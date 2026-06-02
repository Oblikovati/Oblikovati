---
milestone: M11
name: Assembly Modeling & Instancing
status: planned
---

# M11 — Assembly Modeling & Instancing

The assembly environment built on the prototype/flyweight instancing model: one shared `ComponentDefinition`, many `ComponentOccurrence`s with their own transforms, nested via occurrence paths, with context proxies binding definition-space geometry into assembly space. Plus component patterns/mirror, substitution, and the BOM.

## Goals

- An assembly component-definition owning occurrences and structure.
- Component occurrences with placement, grounding, suppression, nesting.
- Context proxies binding part-space geometry into assembly space.
- Component patterns/mirror/copy and substitution.
- A BOM with structure, quantities, and item numbering.

## In scope

- `AssemblyComponentDefinition`; occurrences container.
- `ComponentOccurrence` place/ground/suppress/transform; paths/sub-occurrences.
- `CreateGeometryProxy`; `*Proxy` entities; context definition.
- Component patterns/mirror/copy; substitution.
- `BOM`/`BOMView`/`BOMRow`; structure/quantity/item numbers.

## Out of scope (handled elsewhere)

- Constraints/joints/representations (M12).
- Drawing BOM/parts list (M14).

## Exit criteria

- Placing the same part twice creates two occurrences sharing one definition.
- A face proxy reports assembly-space geometry through the occurrence path.
- A component pattern replicates an occurrence with parametric count.
- The BOM reflects structure and quantities.

## Depends on

M07, M08

## Features

| ID | Feature | PBIs | Summary |
|----|---------|:----:|---------|
| **F01** | [Assembly Component Definition](F01-assembly-definition/_feature.md) | 1 | The assembly content object and occurrence container. |
| **F02** | [Component Occurrences](F02-component-occurrences/_feature.md) | 2 | Instances: placement, grounding, suppression, nesting. |
| **F03** | [Context Proxies](F03-context-proxies/_feature.md) | 1 | Bind definition-space geometry into assembly space. |
| **F04** | [Component Patterns, Mirror & Substitution](F04-component-patterns/_feature.md) | 2 | Replicate, mirror, copy, and substitute components. |
| **F05** | [Bill of Materials](F05-bom/_feature.md) | 2 | BOM structure, quantities, and item numbering. |
