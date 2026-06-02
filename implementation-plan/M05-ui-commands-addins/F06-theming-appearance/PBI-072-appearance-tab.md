---
milestone: M05
feature: F06
pbi: PBI-072
title: Preferences Appearance tab
status: done
estimate: S
---

# PBI-072 — Preferences Appearance tab

**Milestone:** M05 · **Feature:** F06 UI Theming & Appearance

## Goal

Give the user a Preferences ▸ Appearance pane to pick, create, and recolor themes with a
live preview.

## Scope / work

- `head/ui/preferences_window.go`: tabbed window (Sketch Grid + Appearance).
- `head/ui/appearance_tab.go`: theme selector combo; duplicate-to-custom (named); per-token
  `ColorEdit4` grouped by area, disabled for built-ins; Save / Delete for customs; status
  line for action errors.

## API contracts

- None (head-internal; drives `Session` theme methods).

## Acceptance criteria

- Selecting a theme restyles immediately; duplicating a built-in creates an editable custom
  and activates it; editing a swatch updates the UI live; Save persists; reopening the app
  restores the custom theme and selection.

## Depends on

PBI-071.
