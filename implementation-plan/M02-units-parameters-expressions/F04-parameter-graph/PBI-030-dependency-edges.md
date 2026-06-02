---
milestone: M02
feature: F04
pbi: PBI-030
title: Dependency edges & DrivenBy/Dependents queries
status: planned
estimate: M
---

# PBI-030 — Dependency edges & DrivenBy/Dependents queries

**Milestone:** M02 Units, Parameters & Expressions  ·  **Feature:** F04 Parameter Dependency Graph

## Goal

Build and maintain the parameter dependency graph from parsed expressions and expose it via DrivenBy/Dependents.

## Scope / work

- Extract references during parse → edges.
- Maintain edges on edit/rename/delete.
- Expose `Dependents`/`DrivenBy` as collections.

## API contracts (interfaces / enums / collections)

- `Parameter.Dependents`,`Parameter.DrivenBy`,`ObjectCollection`

## Acceptance criteria

- Editing an expression updates edges.
- Renaming a parameter (by id) preserves edges; display label rewrites.

## Depends on

_See feature dependencies._
