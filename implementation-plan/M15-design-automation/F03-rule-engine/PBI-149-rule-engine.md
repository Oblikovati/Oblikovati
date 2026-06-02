---
milestone: M15
feature: F03
pbi: PBI-149
title: Rule engine, triggers & forms
status: planned
estimate: XL
---

# PBI-149 — Rule engine, triggers & forms

**Milestone:** M15 Design Automation: iPart/iAssembly, Tables & iLogic  ·  **Feature:** F03 iLogic Rule Engine

## Goal

Implement a rule engine that executes automation against the public object model, with event-driven triggers, dependency ordering between rules, and a forms framework for parameter input.

## Scope / work

- Rule store/execution host (scripting).
- Triggers via `ChangeManager`/events.
- Rule dependency ordering.
- Forms binding to parameters/properties.

## API contracts (interfaces / enums / collections)

- RuleManager (over `Application` API), `ChangeManager`, `Parameters`

## Acceptance criteria

- A rule fires on a parameter change and reconfigures features/components deterministically.

## Depends on

_See feature dependencies._

## Notes

Build entirely on the public API surface — rules are an automation client of M00-M14, validating the API's completeness.
