---
milestone: M15
feature: F03
name: iLogic Rule Engine
status: planned
---

# M15 · F03 — iLogic Rule Engine

A rule engine that runs user code/rules against the model to automate parameters, feature suppression, component swapping, and properties, driven by event triggers, with a forms layer for end-user input.

## In scope

- Rule definition/execution against the object model.
- Event triggers (parameter/feature/document change).
- Forms; rule ordering/dependencies.

## Out of scope

_None._

## Key API contracts delivered

- (internal) RuleManager over the public API
- ties to `ChangeManager`(M04),`Parameters`(M02)

## Depends on

M02,M04,M08.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-149](PBI-149-rule-engine.md) | Rule engine, triggers & forms |
