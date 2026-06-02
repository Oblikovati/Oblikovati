---
milestone: M14
feature: F01
pbi: PBI-138
title: Drafting standards & style managers
status: planned
estimate: M
---

# PBI-138 — Drafting standards & style managers

**Milestone:** M14 Drawing & Documentation  ·  **Feature:** F01 Drawing Document & Sheets

## Goal

Implement the drafting standards/style system (dimension/text/layer/line styles) governing drawing appearance, with ISO/ANSI presets.

## Scope / work

- `DrawingStylesManager`; standard styles.
- `DimensionStyle`; text/layer/line styles.
- Standard switching.

## API contracts (interfaces / enums / collections)

- `DrawingStylesManager`,`DrawingStandardStyle`,`DimensionStyle`,`DrawingSettings`

## Acceptance criteria

- Switching standard changes dimension/text appearance globally.

## Depends on

_See feature dependencies._
