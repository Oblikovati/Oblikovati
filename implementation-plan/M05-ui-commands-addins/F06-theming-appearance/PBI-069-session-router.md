---
milestone: M05
feature: F06
pbi: PBI-069
title: Session hooks & router theme.* methods
status: done
estimate: S
---

# PBI-069 — Session hooks & router theme.* methods

**Milestone:** M05 · **Feature:** F06 UI Theming & Appearance

## Goal

Hang the theme library off the session and serve the read-only theme methods over the
wire router.

## Scope / work

- `app/appearance.go`: `Session.Themes` / `Theme` / `ThemeRevision` / `LoadThemes` /
  `SetActiveTheme` / `DuplicateTheme` / `DeleteTheme` / `SaveActiveTheme` (persist via the
  attached store; built-in-only & deterministic without one).
- `addin/router/theme.go`: `themeActive` / `themeList`; registered in `router.go`.

## API contracts

- Serves `wire.MethodThemeActive`, `wire.MethodThemeList`.

## Acceptance criteria

- `theme.active` returns the active theme with a hex value for every token; `theme.list`
  flags the active theme; a duplicated custom appears and becomes active.

## Depends on

PBI-067, PBI-068.
