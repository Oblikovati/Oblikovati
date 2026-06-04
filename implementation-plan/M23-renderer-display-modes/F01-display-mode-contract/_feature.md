---
milestone: M23
feature: F01
name: Display-Mode Contract & Style Plumbing
status: planned
---

# M23 · F01 — Display-Mode Contract & Style Plumbing

The seam every other M23 feature slots into: the public `DisplayModeEnum`, the
renderer-internal `VisualStyle` it maps to, the pure **style→passes resolver** that
decides which passes a mode draws, and the Visual Style ribbon gallery that lets a user
pick one. F02/F03/F04 each add their passes *behind* this resolver — they do not touch
the contract or the UI again. See [ADR-0023](../../../architecture/decisions/ADR-0023-viewport-display-modes.md) §1.

## In scope

- `DisplayModeEnum` in `api/types` mirroring `DisplayModeEnum.cs` exactly (8706–8716,
  8707 aliased); `DisplayMode` get/set on `contract.View`/`ClientView`; `api/wire`
  method constants + DTOs; an `api/client` group; `addin/router` handler.
- `renderer.VisualStyle` extended to the full mode set + a pure resolver from a mode to
  its pass set (surface / edge / hidden-edge / npr-style).
- A Visual Style ribbon gallery (`head/ui`) + `CommandDefinition` + interactive select.

## Out of scope

- The passes themselves (F02 PBR, F03 hidden-line, F04 NPR). This feature lands the
  enum, the resolver, and the UI with the three existing styles wired; new modes select
  a still-stub pass set until their feature lands.

## Key API contracts delivered

- `api/types` `DisplayModeEnum` (8706–8716)
- `contract.View.DisplayMode` get/set (and `ClientView`)
- `api/wire` `view.getDisplayMode` / `view.setDisplayMode` + DTOs
- `api/client` view display-mode group

## Depends on

M05 (commands/UI), the existing `renderer.VisualStyle` + `BuildDrawListStyled`
([renderer/drawlist.go](../../../renderer/drawlist.go)).

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-300](PBI-300-display-mode-contract.md) | `DisplayModeEnum` public contract + `View.DisplayMode` get/set |
| [PBI-301](PBI-301-visual-style-resolver.md) | Full `VisualStyle` set + pure style→passes resolver |
| [PBI-302](PBI-302-visual-style-gallery.md) | Visual Style ribbon gallery + command + end-to-end select |
