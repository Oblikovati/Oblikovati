---
milestone: M16
feature: F04
pbi: PBI-156
title: Presentation document, exploded views & tweaks
status: planned
estimate: L
---

# PBI-156 — Presentation document, exploded views & tweaks

**Milestone:** M16 Visualization, Appearances, Styles & Presentations  ·  **Feature:** F04 Presentations & Exploded Views

## Goal

Implement the presentation document with exploded views composed of component tweaks and trails, referencing assembly occurrences by path.

## Scope / work

- `PresentationDocument` from an assembly.
- `PresentationExplodedView` + `Tweak` (translate/rotate) + trails.
- Occurrence-path references.

## API contracts (interfaces / enums / collections)

- `PresentationDocument`,`PresentationExplodedView(s)`,`Tweak(s)`,`PresentationComponent`

## Acceptance criteria

- Components explode along tweaks with visible trails; collapse restores assembled state.

## Depends on

_See feature dependencies._
