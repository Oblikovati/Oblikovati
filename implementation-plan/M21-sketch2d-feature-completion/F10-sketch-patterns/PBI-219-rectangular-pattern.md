---
milestone: M21
feature: F10
pbi: PBI-219
title: Rectangular sketch pattern
status: planned
estimate: M
---

# PBI-219 — Rectangular sketch pattern

**Milestone:** M21  ·  **Feature:** F10 Sketch Patterns

## Goal

Duplicate a sketch selection in a 1- or 2-direction grid as a `SketchRectangularPattern`
following the Definition→Add→Feature triangle.

## Scope / work

- **/source:** `model/sketch/patterns.go` — `SketchRectangularPatternDefinition` (seed
  selection, dir1/dir2 vectors, counts, spacings, suppress list),
  `SketchRectangularPatterns.Add(def)`, `SketchRectangularPattern` whose `.Definition`
  round-trips and regenerates the copies (reusing F09 `Copy`). Serialize round-trip.
- **/api:** `types.SketchPatternKind`; `wire.AddSketchPatternArgs`; `MethodSketchAddPattern`;
  `client.Sketch.Pattern.Rectangular`.
- **UI:** rectangular-pattern tool + `head/ui/sketch_rect_pattern_dialog.go` (counts/spacing/
  directions), ribbon Pattern panel, e2e.

## Acceptance criteria

- Dogfood + UI e2e: a 3×2 pattern yields 6 instances at the right positions; editing the
  count regenerates; editing the seed updates every copy. Round-trip preserved. `make ci` green.

## Depends on

PBI-216 (copy).
