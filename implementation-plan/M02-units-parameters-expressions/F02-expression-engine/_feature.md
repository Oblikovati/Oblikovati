---
milestone: M02
feature: F02
name: Expression Engine
status: planned
---

# M02 · F02 — Expression Engine

Parses and evaluates algebraic expressions (`2 * width + 5 mm`) with unit propagation, built-in functions, and references to other parameters by stable id.

## In scope

- Tokenizer/parser/AST.
- Unit-aware arithmetic & functions.
- Reference resolution by parameter id (not text).

## Out of scope

_None._

## Key API contracts delivered

- (internal) ExpressionEvaluator
- `ExpressionList`

## Depends on

F01.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-025](PBI-025-expression-parser.md) | Expression parser & AST |
| [PBI-026](PBI-026-expression-eval.md) | Unit-aware evaluator with built-in functions |
