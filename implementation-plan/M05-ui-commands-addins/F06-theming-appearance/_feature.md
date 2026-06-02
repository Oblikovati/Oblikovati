---
milestone: M05
feature: F06
name: UI Theming & Appearance
status: in-progress
---

# M05 · F06 — UI Theming & Appearance

A ThemeManager that styles the whole application shell — window, menus, ribbon icons,
viewport 2D overlays, and 3D gizmos — from a curated set of semantic color tokens. Ships
a Light and a Dark built-in; the user can duplicate one into an editable custom theme and
recolor any token with a color picker, with the UI updating live. Custom themes persist
per user as YAML. Excludes 3D body appearance (the future material/appearance subsystem).

See [ADR-0021](../../../architecture/decisions/ADR-0021-ui-theming-semantic-tokens.md).

## In scope

- Semantic token palette (`api/types.ThemeToken`) grouped Chrome / Viewport 2D / Gizmos
  3D / Icons; `Rgba` value type; built-in Dark + Light palettes.
- Read-only public surface: `contract.Theme`, `theme.active` / `theme.list` wire methods,
  typed `client.Theme` group.
- Core `theme/` package: palette, library with active selection + revision counter,
  YAML persistence to the user config dir behind a filesystem seam.
- `Session` integration; `addin/router` `theme.*` handlers.
- Head apply layer: token → Dear ImGui slot binding (by name), overlay/icon color
  refresh, themed viewport + swapchain clear; native `SetStyleColor` / `ColorEdit4` /
  `BeginCombo` verbs.
- Preferences ▸ Appearance tab: theme selector, duplicate, per-token color picker with
  live preview, save, delete.

## Out of scope

- 3D body/material appearance (later subsystem).
- Add-in-authored or add-in-mutated themes (read-only access only).
- Font/spacing/sizing theming (colors only for v1).

## Key API contracts delivered

- `types.ThemeToken`, `types.Rgba`, `types.ThemeKind`
- `contract.Theme`
- `wire.ThemeView`, `wire.ThemeSummary`, `wire.ListThemesResult`,
  `MethodThemeActive`, `MethodThemeList`
- `client.Theme`

## Depends on

F02 (command framework), F03 (ribbon/browser/docking chrome), ADR-0018, ADR-0020.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-066](PBI-066-theme-contract.md) | Theme public contract (api) |
| [PBI-067](PBI-067-core-theme-package.md) | Core theme package: palette, defaults, library |
| [PBI-068](PBI-068-theme-persistence.md) | Custom theme persistence (user config dir) |
| [PBI-069](PBI-069-session-router.md) | Session hooks & router theme.* methods |
| [PBI-070](PBI-070-native-style-verbs.md) | Native ImGui style/picker/combo verbs |
| [PBI-071](PBI-071-theme-apply-layer.md) | Head apply layer & themed overlays |
| [PBI-072](PBI-072-appearance-tab.md) | Preferences Appearance tab |
