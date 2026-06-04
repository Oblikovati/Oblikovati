---
milestone: M21
feature: F06
pbi: PBI-212
title: Ground/offset/align/pattern constraints
status: planned
estimate: M
---

# PBI-212 — Ground/offset/align/pattern constraints

**Milestone:** M21  ·  **Feature:** F06 Geometric Constraints

## Goal

Add the geometric constraints the model lacks: ground, offset, horizontal/vertical align,
and the pattern constraint (consumed by F10 patterns).

## Scope / work

- **/source:** in `model/sketch/constraints_2d.go` (or a new file) add `GroundConstraint`
  (fix all DOF of an entity, user-visible vs internal fix), `OffsetConstraint` (hold two
  curves a fixed offset apart — used by F05 offset), `HorizontalAlignConstraint`/
  `VerticalAlignConstraint` (align two points' axis), `PatternConstraint` (bind a pattern
  member to its seed). Residual/variables + solver wiring; serialize round-trip.
- **/api:** new `GeometricConstraintKind` members + `client.Constrain` helpers.
- **UI:** ground + align tools on the Constrain panel; e2e.

## Acceptance criteria

- Dogfood + UI e2e: ground fixes an entity (DOF→0 for it); align makes two points share an
  axis; offset holds spacing under a drive. Round-trip preserves all. `make ci` green.

## Depends on

PBI-211.
