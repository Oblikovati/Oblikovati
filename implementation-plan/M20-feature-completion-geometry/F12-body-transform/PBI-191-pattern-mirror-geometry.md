---
milestone: M20
feature: F12
pbi: PBI-191
title: Pattern & mirror real duplication
status: planned
estimate: L
---

# PBI-191 — Pattern & mirror real duplication

**Milestone:** M20 Feature Completion & Geometry Parity  ·  **Feature:** F12 Body Transform Features

## Goal

Make the existing Rectangular/Circular/SketchDriven patterns and Mirror emit real transformed copies via the new op.

## Scope / work

Wire `ops.TransformBody` into the pattern/mirror recompute so each active element is a real placed copy (booleaned into the result for join, kept separate for new-body); per-element suppression honored.

## API contracts (interfaces / enums / collections)

- `RectangularPatternFeature`/`CircularPatternFeature`/`SketchDrivenPatternFeature`/`MirrorFeature` real geometry.

## Acceptance criteria

- A 1×3 rectangular pattern of a boss yields three real placed solids at the right pitch
- mirror reflects the source across a plane
- suppressing element 2 removes only it
- recompute.

## Depends on

M07

## Notes

Follows the canonical feature pattern set by Extrude (M08·PBI-092): full
`Definition → Add → Feature(+Proxy)` triangle, `/api` `wire`+`client`+`contract`,
`addin/router` handler, `.obk` round-trip, and executable tests.

## Fix 2026-06-05 — pattern replicates the source feature's TOOL, not the whole body

A pattern/mirror must re-apply the **source feature's own contribution** at each occurrence, not
copy the whole running solid. Copying the body split a patterned cut/join into **N separate
bodies** (a patterned hole gave N solids instead of one body with N holes; a patterned lofted
blade gave 7 bodies, not one fan). Landed:

- `feature.ToolFeature` / `OperationalFeature` (`model/feature/feature.go`): a boolean feature
  exposes its `Operation()` (Cut/Join/Intersect) and, where it builds a discrete prism/sweep
  solid, its `ToolBody()`. The engine's `Input.SourceTool` resolver
  (`engine.go::sourceTool`) prefers the cached tool, else derives the before/after **delta**
  (`sourceDelta`); `nonEmpty` treats an empty difference as "no contribution".
- Pattern replication (`patterns.go::replicate`/`replicateTool`): for a boolean source it
  transforms the source tool per occurrence and re-applies the **same boolean** against the
  running result (one body, N holes/bosses); a boolean source whose tool cannot be recovered
  (a **deferred** feature, e.g. Boss) replicates **nothing** rather than duplicating the body.
- Coverage: **all** boolean tool features implement the contract — Extrude, Revolve, Coil, Rib,
  Emboss, Loft, Sweep (ToolBody) and **Hole** (caches its drill cylinder); **Boss** is
  `OperationalFeature` only (geometry still deferred). Pinned by `toolfeature_test.go`,
  `patterns_test.go` (`TestPatternOfCutKeepsOneBody`/`TestPatternOfJoinMergesIntoOneBody`),
  `hole_boss_test.go::TestPatternOfHoleCutsEachOccurrence`.
