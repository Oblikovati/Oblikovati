---
milestone: M05
feature: F06
pbi: PBI-066
title: Theme public contract (api)
status: done
estimate: S
---

# PBI-066 — Theme public contract (api)

**Milestone:** M05 · **Feature:** F06 UI Theming & Appearance

## Goal

Define the Apache-2.0 public surface for UI themes, once, in `Oblikovati.API`.

## Scope / work

- `api/types`: `Rgba` (0..1 RGBA with `ParseHex`/`Hex`/`Array`), `ThemeToken` string
  enum (Chrome / Viewport 2D / Gizmos 3D / Icons) + `AllThemeTokens()`, `ThemeKind`.
- `api/contract`: `Theme` interface (`Name`, `Kind`, `Color`).
- `api/wire`: `ThemeView`, `ThemeSummary`, `ListThemesResult`, `MethodThemeActive`,
  `MethodThemeList`.
- `api/client`: typed `Theme` group (`Active`, `List`).

## API contracts

- `types.ThemeToken`, `types.Rgba`, `types.ThemeKind`, `contract.Theme`, `wire.ThemeView`,
  `client.Theme`.

## Acceptance criteria

- `go build ./...` + tests green in the api module; hex parse round-trips; token list has
  no duplicates.

## Depends on

ADR-0018, ADR-0021.
