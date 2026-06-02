---
milestone: M15
feature: F01
pbi: PBI-146
title: iPart/iAssembly factory & member generation
status: planned
estimate: L
---

# PBI-146 — iPart/iAssembly factory & member generation

**Milestone:** M15 Design Automation: iPart/iAssembly, Tables & iLogic  ·  **Feature:** F01 iPart/iAssembly Factories

## Goal

Implement the factory that defines a member table (controlling parameters/features/properties/components) and generates member documents, with key columns and custom overrides.

## Scope / work

- `iPartFactory` table model.
- Member doc generation & cache.
- Key columns; custom members; `iAssemblyFactory`.

## API contracts (interfaces / enums / collections)

- `iPartFactory`,`iPartTableRow(s)`,`iPartTableColumn(s)`,`iPartMember(s)`,`iAssemblyFactory`

## Acceptance criteria

- Selecting a row generates the correct member geometry/properties; custom edits persist.

## Depends on

_See feature dependencies._
