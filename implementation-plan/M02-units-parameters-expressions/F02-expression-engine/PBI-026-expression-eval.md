---
milestone: M02
feature: F02
pbi: PBI-026
title: Unit-aware evaluator with built-in functions
status: planned
estimate: M
---

# PBI-026 — Unit-aware evaluator with built-in functions

**Milestone:** M02 Units, Parameters & Expressions  ·  **Feature:** F02 Expression Engine

## Goal

Evaluate the AST with dimensional analysis (reject unit-incompatible operations) and a standard function library (trig, sqrt, min/max…).

## Scope / work

- Dimensional checking.
- Function library.
- Constant folding.

## API contracts (interfaces / enums / collections)

- (internal) evaluator
- `ExpressionList`

## Acceptance criteria

- `sin(30 deg)`=0.5; `1 mm + 1 deg` errors.
- Results are dimensioned quantities.

## Depends on

_See feature dependencies._
