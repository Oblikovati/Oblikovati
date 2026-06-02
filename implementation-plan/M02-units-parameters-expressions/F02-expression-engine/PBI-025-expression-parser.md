---
milestone: M02
feature: F02
pbi: PBI-025
title: Expression parser & AST
status: planned
estimate: M
---

# PBI-025 — Expression parser & AST

**Milestone:** M02 Units, Parameters & Expressions  ·  **Feature:** F02 Expression Engine

## Goal

Implement the grammar, tokenizer, and AST for parameter expressions including units and function calls.

## Scope / work

- Operators, precedence, parentheses.
- Unit literals; function syntax.
- Error reporting with positions.

## API contracts (interfaces / enums / collections)

- (internal) parser/AST

## Acceptance criteria

- Valid expressions parse; malformed ones report position.
- Unit literals parse to dimensioned constants.

## Depends on

_See feature dependencies._
