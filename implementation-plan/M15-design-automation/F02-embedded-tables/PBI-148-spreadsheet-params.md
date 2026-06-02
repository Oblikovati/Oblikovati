---
milestone: M15
feature: F02
pbi: PBI-148
title: Spreadsheet-driven parameters & expression lists
status: planned
estimate: M
---

# PBI-148 — Spreadsheet-driven parameters & expression lists

**Milestone:** M15 Design Automation: iPart/iAssembly, Tables & iLogic  ·  **Feature:** F02 Embedded & Spreadsheet Tables

## Goal

Implement embedded/linked spreadsheet control of parameters with sync, and expression lists for enumerated parameter values.

## Scope / work

- Link spreadsheet cells ↔ parameters.
- Sync direction & refresh.
- `ExpressionList` for pick-lists.

## API contracts (interfaces / enums / collections)

- `ExpressionList(s)`, spreadsheet-link API

## Acceptance criteria

- Editing a linked cell updates the parameter and recomputes.

## Depends on

_See feature dependencies._
