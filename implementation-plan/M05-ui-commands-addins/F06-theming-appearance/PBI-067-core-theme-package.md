---
milestone: M05
feature: F06
pbi: PBI-067
title: Core theme package — palette, defaults, library
status: done
estimate: M
---

# PBI-067 — Core theme package: palette, defaults, library

**Milestone:** M05 · **Feature:** F06 UI Theming & Appearance

## Goal

Implement the GPL theme model: palettes, the built-in Dark/Light themes, and the library
with an active selection and a live-apply revision counter.

## Scope / work

- `theme/token.go`: type aliases + editor metadata (`TokenInfo`, `Group`, `InfoFor`).
- `theme/palette.go`: `Palette` (full snapshot), `Color`/`Clone`/`Hex`/`NewPalette`.
- `theme/defaults.go`: `DefaultDark` (reproduces the pre-theming hardcoded colors) /
  `DefaultLight` / `Builtins`.
- `theme/theme.go`: `Theme` satisfying `contract.Theme`; read-only built-ins; `Duplicate`.
- `theme/library.go`: built-ins + customs, active selection, monotonic `Revision`,
  `SetActive` / `Duplicate` / `Remove` / `EditActiveColor`.

## API contracts

- Implements `contract.Theme` (compile-time asserted).

## Acceptance criteria

- Both built-ins define every token (`TestDefaultsComplete`); built-ins are immutable;
  duplicate is an independent snapshot; select/edit bump the revision; unknown active name
  falls back to Dark.

## Depends on

PBI-066.
