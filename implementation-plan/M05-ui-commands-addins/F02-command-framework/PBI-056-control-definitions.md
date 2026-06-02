---
milestone: M05
feature: F02
pbi: PBI-056
title: ControlDefinitions (button/list/combo/checkbox)
status: planned
estimate: M
---

# PBI-056 — ControlDefinitions (button/list/combo/checkbox)

**Milestone:** M05 Application UI, Commands, Interaction & Add-in Platform  ·  **Feature:** F02 Command & Control Framework

## Goal

Implement the command-control definitions with icons, tooltips, enabled state, and the events that fire on execute.

## Scope / work

- `ButtonDefinition`,`ComboBoxDefinition`,`ListControlDefinition`,`SplitButton`.
- Icon/tooltip/standard sizes; enabled/visible.
- `ControlDefinitionEvents.OnExecute`.

## API contracts (interfaces / enums / collections)

- `ControlDefinitions`,`ControlDefinition`,`ButtonDefinition`,`ComboBoxDefinition`,`ControlDefinitionEvents`

## Acceptance criteria

- A button definition fires OnExecute when invoked.
- Enabled/visible state is honored.

## Depends on

_See feature dependencies._
