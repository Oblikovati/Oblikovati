---
milestone: M23
feature: F01
pbi: PBI-300
title: DisplayModeEnum public contract + View.DisplayMode get/set
status: planned
estimate: M
---

# PBI-300 — `DisplayModeEnum` public contract + `View.DisplayMode` get/set

**Milestone:** M23 Renderer Display-Mode Parity & Realistic PBR  ·  **Feature:** F01 Display-Mode Contract & Style Plumbing

## Goal

Expose the display mode on the public surface so an add-in (and the host UI) can read and
set the viewport's visual style by the exact Inventor `DisplayModeEnum` member.

## Scope / work

- Define `DisplayModeEnum` **once** in `api/types`, mirroring
  `Oblikovati.Contracts/.../Enums/DisplayModeEnum.cs` exactly: members and **stable ids**
  8706–8716, including the `kHiddenEdgeRendering` / `kShadedWithHiddenEdgesRendering`
  alias on 8707. Never renumber (CONVENTIONS naming rule).
- Add `DisplayMode` get/set to `contract.View` (and `ClientView`).
- Add `view.getDisplayMode` / `view.setDisplayMode` method-name constants + JSON DTOs to
  `api/wire`; a typed group to `api/client`.
- In `/source`: alias `type DisplayModeEnum = types.DisplayModeEnum`; implement get/set on
  the view; `var _ contract.View = (*impl.View)(nil)`; wire both methods into
  `addin/router` keyed on the `api/wire` constants. Map the public enum to the internal
  `renderer.VisualStyle` (delegates to the PBI-301 resolver; until then, the three
  existing styles map directly and unimplemented modes return a typed "not yet rendered"
  state rather than a silent fallback).
- SPDX headers on every new file (Apache-2.0 in `/api`, GPL-2.0-only in `/source`).

## API contracts (interfaces / enums / collections)

- `api/types`: `DisplayModeEnum` (8706–8716, 8707 aliased)
- `api/contract`: `View.DisplayMode` get/set; `ClientView.DisplayMode`
- `api/wire`: `view.getDisplayMode`, `view.setDisplayMode` + request/response DTOs
- `api/client`: typed view display-mode methods

## Acceptance criteria

- Dogfood test: for **every** `DisplayModeEnum` member, `api/client` `setDisplayMode`
  then `getDisplayMode` round-trips the same member over the `api/wire` router.
- The 8707 alias resolves to a single canonical member on get (documented), and both
  spellings are accepted on set.
- Compile-time `var _ contract.View = (*impl.View)(nil)` holds; CI's `/api`-must-not-
  import-`/source` check stays green.
- Setting an as-yet-unrendered mode (F02/F03/F04 territory) is a defined, typed result —
  not a panic and not a silent wrong style.

## Depends on

M05 (the view/command surface). First PBI of M23.
