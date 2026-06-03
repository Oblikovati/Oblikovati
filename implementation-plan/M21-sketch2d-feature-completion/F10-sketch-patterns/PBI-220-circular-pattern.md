---
milestone: M21
feature: F10
pbi: PBI-220
title: Circular sketch pattern + pattern constraint
status: planned
estimate: M
---

# PBI-220 — Circular sketch pattern + pattern constraint

**Milestone:** M21  ·  **Feature:** F10 Sketch Patterns

## Goal

Duplicate a sketch selection around a center as a `SketchCircularPattern`, and bind the
members to the seed via `PatternConstraint` so the array stays parametric.

## Scope / work

- **/source:** `model/sketch/patterns.go` — `SketchCircularPatternDefinition` (seed, center,
  count, total/incremental angle, fitted, suppress), `SketchCircularPatterns.Add(def)`,
  `SketchCircularPattern` regenerating rotated copies (reusing F09 `Rotate`); attach the
  `PatternConstraint` from F06. Serialize round-trip.
- **/api:** `SketchPatternKind` circular member; `client.Sketch.Pattern.Circular`.
- **UI:** circular-pattern tool + `head/ui/sketch_circ_pattern_dialog.go` (center/count/angle),
  ribbon Pattern panel, e2e.

## Acceptance criteria

- Dogfood + UI e2e: a count-6 circular pattern over 360° places 6 instances at 60° steps;
  editing the seed updates all; suppress hides one. Round-trip preserved. `make ci` green.

## Depends on

PBI-219, PBI-212 (pattern constraint).
