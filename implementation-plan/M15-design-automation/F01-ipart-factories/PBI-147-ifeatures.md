---
milestone: M15
feature: F01
pbi: PBI-147
title: iFeatures (reusable feature templates)
status: planned
estimate: M
---

# PBI-147 — iFeatures (reusable feature templates)

**Milestone:** M15 Design Automation: iPart/iAssembly, Tables & iLogic  ·  **Feature:** F01 iPart/iAssembly Factories

## Goal

Implement extractable, parameter-driven feature templates (iFeatures) that can be placed into other parts (also used by sheet-metal punches).

## Scope / work

- Extract feature(s) → iFeature with exposed params.
- Place into target with value mapping.

## API contracts (interfaces / enums / collections)

- `iFeature(s)`,`iFeatureDefinition`

## Acceptance criteria

- An extracted iFeature places into another part with parameter prompts.

## Depends on

_See feature dependencies._
