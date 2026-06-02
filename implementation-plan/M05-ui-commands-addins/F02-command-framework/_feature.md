---
milestone: M05
feature: F02
name: Command & Control Framework
status: planned
---

# M05 · F02 — Command & Control Framework

The framework for defining commands (buttons, lists, combos, checkboxes) via `ControlDefinitions`, organizing them in categories, and driving their execute/terminate lifecycle with events.

## In scope

- `CommandManager`, `ControlDefinitions`.
- Button/list/combo/checkbox/spinner control definitions.
- `CommandCategory`; command execute/terminate events; hotkeys.

## Out of scope

_None._

## Key API contracts delivered

- `CommandManager`,`ControlDefinitions`,`ControlDefinition`,`ButtonDefinition`,`ComboBoxDefinition`,`ListControlDefinition`
- `CommandCategory`,`CommandCategories`,`ControlDefinitionEvents`

## Depends on

F01.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-056](PBI-056-control-definitions.md) | ControlDefinitions (button/list/combo/checkbox) |
| [PBI-057](PBI-057-command-manager.md) | CommandManager, categories & interactive command lifecycle |
