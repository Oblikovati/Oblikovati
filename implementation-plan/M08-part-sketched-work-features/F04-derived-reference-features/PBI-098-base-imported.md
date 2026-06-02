---
milestone: M08
feature: F04
pbi: PBI-098
title: Non-parametric base & imported geometry features
status: planned
estimate: M
---

# PBI-098 — Non-parametric base & imported geometry features

**Milestone:** M08 Part Modeling: Sketched & Work Features  ·  **Feature:** F04 Derived & Reference Features

## Goal

Implement non-parametric base bodies (from import/translation, M17) as feature-tree participants that can be edited downstream.

## Scope / work

- `NonParametricBaseFeature` wrapping imported bodies.
- `ImportedComponent` definition link.
- Downstream-editability.

## API contracts (interfaces / enums / collections)

- `NonParametricBaseFeature(s)`,`ImportedComponent(s)`,`ImportedComponentDefinition`

## Acceptance criteria

- An imported solid appears as a base feature editable by later features.

## Depends on

_See feature dependencies._
