---
milestone: M21
feature: F09
pbi: PBI-218
title: Mirror & delete
status: planned
estimate: M
---

# PBI-218 — Mirror & delete

**Milestone:** M21  ·  **Feature:** F09 Editing Operations

## Goal

Mirror a selection about a sketch line (creating symmetric copies tied by symmetry
constraints) and delete entities with cascading constraint cleanup.

## Scope / work

- **/source:** `model/sketch/edit_ops.go` — `Mirror(sel, line)` reflects the selection and
  adds `SymmetryConstraint`s (reusing F06); `Delete(entities)` removes entities and the
  constraints/dimensions that reference them. Serialize round-trip.
- **/api:** `MethodSketchMirror`, `wire.MirrorSketchArgs`; `client.Sketch.Mirror`;
  delete via the existing entity handle.
- **UI:** mirror tool (select geometry + mirror line), ribbon Pattern panel; delete via
  the standard delete action; e2e.

## Acceptance criteria

- Dogfood + UI e2e: mirroring across a line produces a reflected copy with symmetry
  constraints (driving one side moves the other); deleting a line removes its constraints.
  Round-trip preserved. `make ci` green.

## Depends on

PBI-212 (symmetry/align constraints).
