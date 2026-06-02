---
milestone: M16
feature: F01
pbi: PBI-152
title: Appearance source resolution & overrides
status: planned
estimate: M
---

# PBI-152 — Appearance source resolution & overrides

**Milestone:** M16 Visualization, Appearances, Styles & Presentations  ·  **Feature:** F01 Appearances & Materials

## Goal

Implement the appearance-source precedence (body/feature/face/occurrence overrides vs material default) that resolves what each entity renders.

## Scope / work

- `AppearanceSourceType` per entity level.
- Override get/set on face/feature/body/occurrence.
- `GetRenderStyle`/`SetRenderStyle`.

## API contracts (interfaces / enums / collections)

- `AppearanceSourceTypeEnum`,`RenderStyle`,`StyleSourceTypeEnum`

## Acceptance criteria

- A face override wins over the part material; clearing reverts to source.

## Depends on

_See feature dependencies._
