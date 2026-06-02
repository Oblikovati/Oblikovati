---
milestone: M15
name: Design Automation: iPart/iAssembly, Tables & iLogic
status: planned
---

# M15 — Design Automation: iPart/iAssembly, Tables & iLogic

Configuration and rule-based automation: iPart/iAssembly factories that generate family members from a table, embedded/Excel-driven tables, and the iLogic-style rule engine that automates parameters, features, and components with event triggers and forms. This layer treats the Definition objects (M08) as the serializable parametric records it drives.

## Goals

- iPart/iAssembly factories generating members from tables.
- Embedded and spreadsheet-linked parameter tables.
- A rule engine automating model state with triggers and forms.
- A standard-parts content library and template management.

## In scope

- `iPartFactory`/`iAssemblyFactory`; rows/columns; member generation.
- Embedded tables; Excel linkage; `ExpressionList`.
- Rule engine; triggers; forms.
- Content center; templates/design data.

## Out of scope (handled elsewhere)

- The parameter engine itself (M02).
- UI for rule editing beyond the framework (M05).

## Exit criteria

- An iPart factory generates distinct members selected by key columns.
- A rule fires on a parameter change and reconfigures the model.
- A spreadsheet drives parameters bidirectionally.

## Depends on

M08, M11

## Features

| ID | Feature | PBIs | Summary |
|----|---------|:----:|---------|
| **F01** | [iPart/iAssembly Factories](F01-ipart-factories/_feature.md) | 2 | Table-driven part/assembly family generation. |
| **F02** | [Embedded & Spreadsheet Tables](F02-embedded-tables/_feature.md) | 1 | Parameter tables and Excel linkage. |
| **F03** | [iLogic Rule Engine](F03-rule-engine/_feature.md) | 1 | Rules automating parameters, features, components; triggers; forms. |
| **F04** | [Content Center & Templates](F04-content-center/_feature.md) | 1 | Standard-parts library and template/design-data management. |
