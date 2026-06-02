---
milestone: M15
feature: F04
pbi: PBI-150
title: Content center library & templates
status: planned
estimate: L
---

# PBI-150 — Content center library & templates

**Milestone:** M15 Design Automation: iPart/iAssembly, Tables & iLogic  ·  **Feature:** F04 Content Center & Templates

## Goal

Implement a content library of standard-part families with on-demand member generation and the template/design-data management for new documents.

## Scope / work

- Family library schema.
- On-demand standard-part generation (reuses iPart factories).
- Template & design-data library paths.

## API contracts (interfaces / enums / collections)

- ContentCenter API,`iPartFactory`,`FileManager`

## Acceptance criteria

- Placing a standard fastener generates the correct member from the library.

## Depends on

_See feature dependencies._
