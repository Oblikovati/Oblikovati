---
milestone: M00
feature: F04
pbi: PBI-013
title: SoftwareVersion, locale & options scaffold
status: planned
estimate: S
---

# PBI-013 — SoftwareVersion, locale & options scaffold

**Milestone:** M00 Platform Foundation & Interop  ·  **Feature:** F04 Application Session & Lifecycle

## Goal

Expose version/locale identity and the application-level options objects that later milestones populate.

## Scope / work

- `SoftwareVersion`, `Locale`, `LanguageName`, `UserName`.
- Empty-but-wired options objects (General/Sketch/Part/Assembly).

## API contracts (interfaces / enums / collections)

- `SoftwareVersion`
- `GeneralOptions`,`SketchOptions`,`PartOptions`,`AssemblyOptions`

## Acceptance criteria

- Version/locale are queryable.
- Options objects exist and persist their (initially few) settings.

## Depends on

_See feature dependencies._
