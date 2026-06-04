---
milestone: M21
feature: F10
name: Sketch Patterns
status: done
---

# M21 · F10 — Sketch Patterns

Duplicate sketch geometry in a grid or around a center: `SketchRectangularPatterns` and
`SketchCircularPatterns`, each following the Definition→Add→Feature triangle, with
`PatternConstraint` tying members to the seed so editing the seed updates every copy.

## In scope

- `SketchRectangularPattern` (two directions, counts, spacings) + Definition/Add/Feature.
- `SketchCircularPattern` (center, count, angle, fitted/incremental) + triangle.
- `PatternConstraint` binding members to the seed (suppress individual members).

## Out of scope

- Feature-level patterns (M09); mirror (F09).

## Key API contracts delivered

- `types.SketchPatternKind`; `MethodSketchAddPattern`
- `client.Sketch.Pattern.{Rectangular,Circular}`

## Depends on

F09 (copy), F06 (pattern constraint).

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-219](PBI-219-rectangular-pattern.md) | Rectangular sketch pattern |
| [PBI-220](PBI-220-circular-pattern.md) | Circular sketch pattern + pattern constraint |
