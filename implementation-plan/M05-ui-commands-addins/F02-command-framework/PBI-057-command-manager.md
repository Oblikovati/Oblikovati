---
milestone: M05
feature: F02
pbi: PBI-057
title: CommandManager, categories & interactive command lifecycle
status: planned
estimate: M
---

# PBI-057 — CommandManager, categories & interactive command lifecycle

**Milestone:** M05 Application UI, Commands, Interaction & Add-in Platform  ·  **Feature:** F02 Command & Control Framework

## Goal

Implement command registration/lookup, categories, and the start/stop of interactive commands tied to selection/interaction.

## Scope / work

- `CommandManager` register/lookup; `CommandCategory`.
- Interactive command start/terminate.
- Hotkey binding.

## API contracts (interfaces / enums / collections)

- `CommandManager`,`CommandCategory`,`CommandCategories`,`Command`,`CommandControl`

## Acceptance criteria

- A command starts, drives an interaction, commits in a transaction, and terminates cleanly.

## Depends on

_See feature dependencies._
