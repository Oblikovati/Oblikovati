---
milestone: M05
feature: F06
pbi: PBI-071
title: Head apply layer & themed overlays
status: done
estimate: M
---

# PBI-071 — Head apply layer & themed overlays

**Milestone:** M05 · **Feature:** F06 UI Theming & Appearance

## Goal

Drive the whole shell's colors from the active theme, live.

## Scope / work

- `head/ui/theme_apply.go`: centralizes the overlay/icon color vars (seeded from Dark);
  `chromeBinding` (token → ImGui slot names); `applyThemeIfChanged` (re-push style +
  refresh vars + viewport clear when the revision changes); `WindowClearColor`.
- Remove the hardcoded color vars from `grid_overlay.go`, `sketch_overlay.go`,
  `plane_overlay.go`, `dimension_overlay.go`, `highlight.go`, `snap_glyph.go`,
  `chrome.go`, `icon_cache.go`.
- `chrome.go` calls `applyThemeIfChanged`; `main.go` clears to `WindowClearColor`.

## API contracts

- None (head-internal).

## Acceptance criteria

- Default look unchanged from before theming; switching Dark↔Light recolors chrome,
  overlays, gizmos, icons, and both clears next frame. Each ImGui slot is bound by exactly
  one token; overlay vars read the active theme (unit-tested without a window).

## Depends on

PBI-069, PBI-070.
