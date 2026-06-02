---
milestone: M16
feature: F02
pbi: PBI-153
title: Style manager, color/lighting styles & libraries
status: planned
estimate: M
---

# PBI-153 — Style manager, color/lighting styles & libraries

**Milestone:** M16 Visualization, Appearances, Styles & Presentations  ·  **Feature:** F02 Styles & Standards

## Goal

Implement the style manager with color/lighting/material styles, style libraries, and style-change events.

## Scope / work

- `StyleManager` registry & cascade.
- `ColorStyle`/`LightingStyle`.
- Library load/save; `StyleEvents`.

## API contracts (interfaces / enums / collections)

- `StyleManager`,`ColorStyle(s)`,`LightingStyle(s)`,`StyleEvents`

## Acceptance criteria

- A style edit propagates to all consumers; library styles import.

## Depends on

_See feature dependencies._
