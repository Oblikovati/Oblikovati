# ADR-0021 — UI theming via semantic tokens (head applies, core owns state)

**Status:** accepted (user decision, 2026-06-02) · **Relates to:**
[ADR-0004](ADR-0004-ui-imgui.md) (Dear ImGui head), [ADR-0018](ADR-0018-apache-api-contract-module.md)
(API contract split), [ADR-0020](ADR-0020-yaml-git-friendly-document-format.md) (YAML files).

## Context

The UI had no theming. Colors were hardcoded as package-level vars scattered across
`head/ui/*.go` (icon tint, grid/plane/sketch/dimension/selection/preview colors), the
window background was a literal `EndFrame(0.12, 0.13, 0.16)`, and the Dear ImGui chrome
used its default style untouched. There was no preference persistence to disk at all.

We want a ThemeManager that styles the whole shell — window, menus, icons, viewport 2D
overlays, and 3D gizmos — ships a Light and a Dark built-in, lets a user duplicate one
into an editable custom theme with a color picker, updates the UI **live** while editing,
and stores custom themes per user. Theming does **not** cover 3D body appearance, which
the future material/appearance subsystem owns.

## Decision

1. **Semantic tokens, not raw ImGui slots.** A curated ~33-token vocabulary
   (`api/types.ThemeToken`, grouped Chrome / Viewport 2D / Gizmos 3D / Icons) is the
   user-facing palette. One token may drive several Dear ImGui color slots (e.g. the
   accent feeds the active tab, checkmark, slider grab, and selection). This keeps the
   editor small and themes coherent, versus exposing ~55 `ImGuiCol_` slots.

2. **Exposed on the public API (ADR-0018 two-part work).** The token enum + `Rgba` value
   type live in `api/types`; a read-only `Theme` interface in `api/contract`;
   `theme.active` / `theme.list` methods + DTOs in `api/wire`; a typed group in
   `api/client`. So an add-in can read the host's active theme and match it.

3. **The core owns theme state; the head applies it.** Because the wire router
   (`addin/router`, GPL app module) cannot import `/head`, the active theme lives on the
   `*app.Session` (a `theme.Library`, alongside `GridSettings`). The head reads it to
   style Dear ImGui and the overlays; the router reads it to serve `theme.*`. The pure-Go
   `theme/` package (palette, defaults, library, IO) has no cgo/ImGui dependency and is
   fully headless-tested.

4. **Full-snapshot custom themes.** Duplicating a built-in copies every token value into
   the custom theme; it is self-contained (no base inheritance to resolve) and keeps its
   look even if a built-in's defaults later change.

5. **YAML files in the user config dir.** Customs persist as readable YAML under
   `os.UserConfigDir()/oblikovati/themes/*.yaml`, the selected theme under
   `.../preferences.yaml`, through the existing `persistence/yamlcodec` seam (ADR-0020),
   behind a `theme.FileSystem` interface (in-memory fake in tests).

6. **Live apply via a revision counter.** `theme.Library` bumps a monotonic revision on
   any select/edit/add/remove. `head/ui` re-pushes the chrome colors into Dear ImGui's
   persistent style (set once per change, not per widget) and refreshes the overlay/icon
   color vars and the viewport clear color only when the revision changes — so a theme
   switch or a color-picker drag restyles the whole UI on the next frame.

## Consequences

- New work touching UI color must add a token (in `api/types`, with a default in both
  built-in palettes — guarded by `TestDefaultsComplete`) rather than a hardcoded color.
- The head maps tokens → ImGui slots by **name** (`native.SetStyleColor("WindowBg", …)`),
  so the binding survives `ImGuiCol_` enum reordering between Dear ImGui versions.
- Themes are presentation-only and per-user; they are not part of a document and never
  travel in a `.obk` file.
- Add-in theme access is read-only for now; an add-in cannot define or mutate host themes.
