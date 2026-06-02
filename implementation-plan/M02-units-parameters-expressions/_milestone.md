---
milestone: M02
name: Units, Parameters & Expressions
status: planned
---

# M02 — Units, Parameters & Expressions

The parametric variable layer: a total unit system in canonical database units, a unit-aware expression engine, the parameter model (model/user/reference/derived/table parameters with tolerance and precision), and the parameter dependency graph that drives recompute. This is the most user-facing parametric surface and is ruinous to retrofit, so it comes before any feature work.

## Goals

- All quantities dimensioned and stored in database units; display in user units.
- A unit-aware expression evaluator with functions and parameter references.
- A full `Parameter` model with tolerance, precision, and parameter types.
- A cyclic-safe dependency DAG driving dirty-propagation recompute.

## In scope

- `UnitsTypeEnum`, conversion, database-unit storage.
- Expression parsing/evaluation with units.
- `Parameter`/`Parameters` model & types.
- `DrivenBy`/`Dependents` graph, cycle detection, `ExpressionList`.

## Out of scope (handled elsewhere)

- Feature recompute engine (M08) consumes this graph.
- iPart tables (M15).

## Exit criteria

- A parameter set with interdependent expressions evaluates correctly and rejects cycles.
- Changing a parameter dirties exactly its transitive dependents.
- Values store in database units and display in user units.

## Depends on

M00

## Features

| ID | Feature | PBIs | Summary |
|----|---------|:----:|---------|
| **F01** | [Unit System](F01-unit-system/_feature.md) | 2 | Dimensioned quantities, database units, and user-unit display. |
| **F02** | [Expression Engine](F02-expression-engine/_feature.md) | 2 | Unit-aware parser/evaluator for parameter expressions. |
| **F03** | [Parameter Model](F03-parameter-model/_feature.md) | 3 | Parameters with types, tolerance, precision, and the value/expression/model-value triad. |
| **F04** | [Parameter Dependency Graph](F04-parameter-graph/_feature.md) | 3 | The expression DAG: DrivenBy/Dependents, cycle detection, dirty-propagation. |
