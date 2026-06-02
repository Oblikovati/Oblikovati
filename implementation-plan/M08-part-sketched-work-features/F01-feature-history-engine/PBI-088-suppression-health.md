---
milestone: M08
feature: F01
pbi: PBI-088
title: Suppression, conditional suppression & health
status: planned
estimate: M
---

# PBI-088 — Suppression, conditional suppression & health

**Milestone:** M08 Part Modeling: Sketched & Work Features  ·  **Feature:** F01 Feature History Engine

## Goal

Implement feature suppression (skip during evaluation), expression-driven conditional suppression, and the health-status surface.

## Scope / work

- `Suppressed` toggle.
- `SetSuppressionCondition(param,comparison,expr)`/`Get`.
- Health states & propagation.

## API contracts (interfaces / enums / collections)

- `PartFeature.Suppressed`,`SetSuppressionCondition`,`ComparisonTypeEnum`,`HealthStatusEnum`

## Acceptance criteria

- A conditionally-suppressed feature toggles as its driving expression crosses the threshold.

## Depends on

_See feature dependencies._
